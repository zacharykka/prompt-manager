package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/zacharykka/prompt-manager/internal/config"
	domain "github.com/zacharykka/prompt-manager/internal/domain"
	authutil "github.com/zacharykka/prompt-manager/pkg/auth"
)

const (
	providerGitHub      = "github"
	tokenTypeOAuthState = "oauth_state"
	gitHubUserAgent     = "prompt-manager-oauth"
)

type gitHubUserInfo struct {
	ID        string
	Login     string
	Email     string
	AvatarURL string
}

// Service 封装认证逻辑。
type Service struct {
	repos            *domain.Repositories
	cfg              config.AuthConfig
	nowFn            func() time.Time
	httpClient       *http.Client
	githubAuthURL    string
	githubTokenURL   string
	githubAPIBaseURL string
}

// Tokens 表示访问令牌与刷新令牌。
type Tokens struct {
	AccessToken           string    `json:"access_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshToken          string    `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
}

// Option 定义 Service 可选项。
type Option func(*Service)

// WithHTTPClient 自定义 HTTP Client（用于注入测试客户端）。
func WithHTTPClient(client *http.Client) Option {
	return func(s *Service) {
		if client != nil {
			s.httpClient = client
		}
	}
}

// WithGitHubEndpoints 自定义 GitHub OAuth 端点，便于测试或代理。
func WithGitHubEndpoints(authURL, tokenURL, apiBaseURL string) Option {
	return func(s *Service) {
		if authURL != "" {
			s.githubAuthURL = authURL
		}
		if tokenURL != "" {
			s.githubTokenURL = tokenURL
		}
		if apiBaseURL != "" {
			s.githubAPIBaseURL = apiBaseURL
		}
	}
}

// NewService 创建认证服务。
func NewService(repos *domain.Repositories, cfg config.AuthConfig, opts ...Option) *Service {
	svc := &Service{
		repos:            repos,
		cfg:              cfg,
		nowFn:            time.Now,
		httpClient:       &http.Client{Timeout: 10 * time.Second},
		githubAuthURL:    "https://github.com/login/oauth/authorize",
		githubTokenURL:   "https://github.com/login/oauth/access_token",
		githubAPIBaseURL: "https://api.github.com",
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// WithClock 允许注入自定义时间函数，便于测试。
func (s *Service) WithClock(now func() time.Time) {
	if now != nil {
		s.nowFn = now
	}
}

// Register 创建新用户。
func (s *Service) Register(ctx context.Context, email, password, role string) (*domain.User, error) {
	email = normalizeEmail(email)
	if email == "" || password == "" {
		return nil, ErrInvalidInput
	}

	if _, err := s.repos.Users.GetByEmail(ctx, email); err == nil {
		return nil, ErrUserExists
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	hash, err := authutil.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		ID:             uuid.NewString(),
		Email:          email,
		HashedPassword: hash,
		Role:           normalizedRole(role),
		Status:         "active",
	}

	if err := s.repos.Users.Create(ctx, user); err != nil {
		return nil, err
	}

	created, err := s.repos.Users.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return created, nil
}

// Login 校验用户凭证并返回令牌。

func (s *Service) Login(ctx context.Context, email, password string) (*Tokens, *domain.User, error) {
	email = normalizeEmail(email)
	if email == "" || password == "" {
		return nil, nil, ErrInvalidCredentials
	}

	user, err := s.repos.Users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	if user.Status != "active" {
		return nil, nil, ErrUserDisabled
	}

	if !authutil.VerifyPassword(user.HashedPassword, password) {
		return nil, nil, ErrInvalidCredentials
	}

	if err := s.repos.Users.UpdateLastLogin(ctx, user.ID); err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, nil, err
	}

	tokens, err := s.issueTokens(user)
	if err != nil {
		return nil, nil, err
	}

	return tokens, user, nil
}

// Refresh 根据刷新令牌生成新令牌。
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*Tokens, *domain.User, error) {
	claims, err := authutil.ParseToken(refreshToken, s.cfg.RefreshTokenSecret)
	if err != nil {
		return nil, nil, ErrTokenInvalid
	}

	if claims.TokenType != "refresh" {
		return nil, nil, ErrTokenInvalid
	}

	user, err := s.repos.Users.GetByEmail(ctx, claims.Subject)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, ErrTokenInvalid
		}
		return nil, nil, err
	}

	tokens, err := s.issueTokens(user)
	if err != nil {
		return nil, nil, err
	}

	return tokens, user, nil
}

// GitHubAuthorizeURL 构造 GitHub OAuth 授权地址。
func (s *Service) GitHubAuthorizeURL(redirectURI string) (string, error) {
	if !s.cfg.GitHub.Enabled {
		return "", ErrOAuthDisabled
	}

	finalRedirect, err := s.normalizeRedirectURI(redirectURI)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrOAuthStateInvalid, err)
	}

	state, err := s.generateOAuthState(providerGitHub, finalRedirect)
	if err != nil {
		return "", err
	}

	query := url.Values{}
	query.Set("client_id", s.cfg.GitHub.ClientID)
	query.Set("redirect_uri", s.cfg.GitHub.RedirectURL)
	if len(s.cfg.GitHub.Scopes) > 0 {
		query.Set("scope", strings.Join(s.cfg.GitHub.Scopes, " "))
	}
	query.Set("state", state)
	query.Set("allow_signup", "false")

	return fmt.Sprintf("%s?%s", s.githubAuthURL, query.Encode()), nil
}

// HandleGitHubCallback 处理 GitHub OAuth 回调并返回本地令牌。
func (s *Service) HandleGitHubCallback(ctx context.Context, code, state string) (*Tokens, *domain.User, string, error) {
	if !s.cfg.GitHub.Enabled {
		return nil, nil, "", ErrOAuthDisabled
	}
	code = strings.TrimSpace(code)
	state = strings.TrimSpace(state)
	if code == "" || state == "" {
		return nil, nil, "", ErrOAuthStateInvalid
	}

	provider, finalRedirect, err := s.parseOAuthState(state)
	if err != nil {
		return nil, nil, "", ErrOAuthStateInvalid
	}
	if provider != providerGitHub {
		return nil, nil, "", ErrOAuthStateInvalid
	}
	if finalRedirect != "" {
		if finalRedirect, err = s.normalizeRedirectURI(finalRedirect); err != nil {
			return nil, nil, "", fmt.Errorf("%w: %v", ErrOAuthStateInvalid, err)
		}
	}

	token, err := s.exchangeGitHubCode(ctx, code, state)
	if err != nil {
		return nil, nil, "", err
	}

	ghUser, err := s.fetchGitHubUser(ctx, token)
	if err != nil {
		return nil, nil, "", err
	}

	email := strings.TrimSpace(ghUser.Email)
	if email == "" {
		email, err = s.fetchPrimaryGitHubEmail(ctx, token)
		if err != nil {
			return nil, nil, "", err
		}
	}

	if err := s.ensureGitHubOrgAccess(ctx, token); err != nil {
		return nil, nil, "", err
	}

	providerUserID := ghUser.ID
	if providerUserID == "" {
		return nil, nil, "", ErrOAuthExchangeFailed
	}

	identity, err := s.repos.UserIdentities.GetByProviderAndExternalID(ctx, providerGitHub, providerUserID)
	var user *domain.User
	if err == nil {
		user, err = s.repos.Users.GetByID(ctx, identity.UserID)
		if err != nil {
			return nil, nil, "", err
		}
	} else if errors.Is(err, domain.ErrNotFound) {
		user, err = s.findOrCreateUserByEmail(ctx, email)
		if err != nil {
			return nil, nil, "", err
		}

		login := strings.TrimSpace(ghUser.Login)
		avatar := strings.TrimSpace(ghUser.AvatarURL)
		identity := &domain.UserIdentity{
			ID:             uuid.NewString(),
			UserID:         user.ID,
			Provider:       providerGitHub,
			ProviderUserID: providerUserID,
		}
		if login != "" {
			identity.ProviderLogin = &login
		}
		if avatar != "" {
			identity.AvatarURL = &avatar
		}

		if err := s.repos.UserIdentities.Create(ctx, identity); err != nil {
			return nil, nil, "", err
		}
	} else {
		return nil, nil, "", err
	}

	if user.Status != "active" {
		return nil, nil, "", ErrUserDisabled
	}

	if err := s.repos.Users.UpdateLastLogin(ctx, user.ID); err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, nil, "", err
	}

	tokens, err := s.issueTokens(user)
	if err != nil {
		return nil, nil, "", err
	}

	return tokens, user, finalRedirect, nil
}

func (s *Service) issueTokens(user *domain.User) (*Tokens, error) {
	now := s.nowFn()
	accessTTL := s.cfg.AccessTokenTTL
	if accessTTL <= 0 {
		accessTTL = 15 * time.Minute
	}
	refreshTTL := s.cfg.RefreshTokenTTL
	if refreshTTL <= 0 {
		refreshTTL = 30 * 24 * time.Hour
	}

	accessClaims := authutil.Claims{
		UserID:    user.ID,
		Role:      user.Role,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:  user.Email,
			Issuer:   "prompt-manager",
			Audience: []string{"prompt-manager"},
		},
	}

	accessToken, err := authutil.GenerateToken(s.cfg.AccessTokenSecret, accessTTL, accessClaims)
	if err != nil {
		return nil, err
	}

	refreshClaims := authutil.Claims{
		UserID:    user.ID,
		Role:      user.Role,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:  user.Email,
			Issuer:   "prompt-manager",
			Audience: []string{"prompt-manager"},
		},
	}

	refreshToken, err := authutil.GenerateToken(s.cfg.RefreshTokenSecret, refreshTTL, refreshClaims)
	if err != nil {
		return nil, err
	}

	tokens := &Tokens{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  now.Add(accessTTL),
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: now.Add(refreshTTL),
	}
	return tokens, nil
}

func (s *Service) generateOAuthState(provider, redirectURI string) (string, error) {
	metadata := map[string]string{
		"provider": provider,
	}
	if redirectURI != "" {
		metadata["redirect_uri"] = redirectURI
	}
	metadata["nonce"] = uuid.NewString()

	claims := authutil.Claims{
		TokenType: tokenTypeOAuthState,
		Metadata:  metadata,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:  "prompt-manager",
			Subject: provider,
			Audience: []string{
				"prompt-manager",
			},
		},
	}

	return authutil.GenerateToken(s.cfg.AccessTokenSecret, s.cfg.GitHub.StateTTL, claims)
}

func (s *Service) parseOAuthState(state string) (string, string, error) {
	claims, err := authutil.ParseToken(state, s.cfg.AccessTokenSecret)
	if err != nil {
		return "", "", err
	}
	if claims.TokenType != tokenTypeOAuthState {
		return "", "", ErrOAuthStateInvalid
	}
	provider := strings.TrimSpace(claims.RegisteredClaims.Subject)
	redirect := ""
	if claims.Metadata != nil {
		redirect = strings.TrimSpace(claims.Metadata["redirect_uri"])
	}
	return provider, redirect, nil
}

func (s *Service) normalizeRedirectURI(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid redirect_uri: %w", err)
	}
	if !u.IsAbs() {
		return "", fmt.Errorf("invalid redirect_uri: must be absolute URL")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("invalid redirect_uri: unsupported scheme")
	}
	return u.String(), nil
}

func (s *Service) exchangeGitHubCode(ctx context.Context, code, state string) (string, error) {
	form := url.Values{}
	form.Set("client_id", s.cfg.GitHub.ClientID)
	form.Set("client_secret", s.cfg.GitHub.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", s.cfg.GitHub.RedirectURL)
	form.Set("state", state)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.githubTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", gitHubUserAgent)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrOAuthExchangeFailed, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", fmt.Errorf("%w: read body", ErrOAuthExchangeFailed)
	}

	var payload struct {
		AccessToken      string `json:"access_token"`
		Scope            string `json:"scope"`
		TokenType        string `json:"token_type"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("%w: decode response", ErrOAuthExchangeFailed)
	}

	if resp.StatusCode >= 400 || payload.Error != "" {
		reason := strings.TrimSpace(payload.ErrorDescription)
		if reason == "" {
			reason = resp.Status
		}
		return "", fmt.Errorf("%w: %s", ErrOAuthExchangeFailed, reason)
	}
	if payload.AccessToken == "" {
		return "", fmt.Errorf("%w: empty access token", ErrOAuthExchangeFailed)
	}
	return payload.AccessToken, nil
}

func (s *Service) fetchGitHubUser(ctx context.Context, accessToken string) (*gitHubUserInfo, error) {
	resp, err := s.doGitHubRequest(ctx, http.MethodGet, "/user", accessToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: fetch user profile", ErrOAuthExchangeFailed)
	}

	var payload struct {
		ID        json.Number `json:"id"`
		Login     string      `json:"login"`
		Email     string      `json:"email"`
		AvatarURL string      `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("%w: decode user profile", ErrOAuthExchangeFailed)
	}

	id := strings.TrimSpace(payload.ID.String())
	if id == "" || id == "0" {
		return nil, fmt.Errorf("%w: invalid user id", ErrOAuthExchangeFailed)
	}

	return &gitHubUserInfo{
		ID:        id,
		Login:     strings.TrimSpace(payload.Login),
		Email:     strings.TrimSpace(payload.Email),
		AvatarURL: strings.TrimSpace(payload.AvatarURL),
	}, nil
}

func (s *Service) fetchPrimaryGitHubEmail(ctx context.Context, accessToken string) (string, error) {
	resp, err := s.doGitHubRequest(ctx, http.MethodGet, "/user/emails", accessToken)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("%w: fetch emails", ErrOAuthExchangeFailed)
	}

	var entries []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return "", fmt.Errorf("%w: decode emails", ErrOAuthExchangeFailed)
	}

	var candidate string
	for _, entry := range entries {
		if !entry.Verified {
			continue
		}
		email := strings.TrimSpace(entry.Email)
		if email == "" {
			continue
		}
		if entry.Primary {
			return email, nil
		}
		if candidate == "" {
			candidate = email
		}
	}

	if candidate != "" {
		return candidate, nil
	}
	return "", ErrOAuthEmailMissing
}

func (s *Service) ensureGitHubOrgAccess(ctx context.Context, accessToken string) error {
	if len(s.cfg.GitHub.AllowedOrgs) == 0 {
		return nil
	}

	orgs, err := s.fetchGitHubOrgs(ctx, accessToken)
	if err != nil {
		return err
	}
	if len(orgs) == 0 {
		return ErrOAuthOrgUnauthorized
	}

	allowed := make(map[string]struct{}, len(s.cfg.GitHub.AllowedOrgs))
	for _, org := range s.cfg.GitHub.AllowedOrgs {
		name := strings.ToLower(strings.TrimSpace(org))
		if name != "" {
			allowed[name] = struct{}{}
		}
	}

	for _, org := range orgs {
		if _, ok := allowed[strings.ToLower(org)]; ok {
			return nil
		}
	}
	return ErrOAuthOrgUnauthorized
}

func (s *Service) fetchGitHubOrgs(ctx context.Context, accessToken string) ([]string, error) {
	resp, err := s.doGitHubRequest(ctx, http.MethodGet, "/user/orgs", accessToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: fetch orgs", ErrOAuthExchangeFailed)
	}

	var payload []struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("%w: decode orgs", ErrOAuthExchangeFailed)
	}

	var orgs []string
	for _, item := range payload {
		name := strings.TrimSpace(item.Login)
		if name != "" {
			orgs = append(orgs, name)
		}
	}
	return orgs, nil
}

func (s *Service) doGitHubRequest(ctx context.Context, method, path, accessToken string) (*http.Response, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("%w: missing access token", ErrOAuthExchangeFailed)
	}
	endpoint := s.githubAPIBaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "token "+accessToken)
	req.Header.Set("User-Agent", gitHubUserAgent)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthExchangeFailed, err)
	}
	return resp, nil
}

func (s *Service) findOrCreateUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	normalized := normalizeEmail(email)
	if normalized == "" {
		return nil, ErrOAuthEmailMissing
	}

	user, err := s.repos.Users.GetByEmail(ctx, normalized)
	if err == nil {
		return user, nil
	}
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	randomSecret := uuid.NewString() + uuid.NewString()
	hash, err := authutil.HashPassword(randomSecret)
	if err != nil {
		return nil, err
	}

	user = &domain.User{
		ID:             uuid.NewString(),
		Email:          normalized,
		HashedPassword: hash,
		Role:           "viewer",
		Status:         "active",
	}

	if err := s.repos.Users.Create(ctx, user); err != nil {
		existing, lookupErr := s.repos.Users.GetByEmail(ctx, normalized)
		if lookupErr == nil {
			return existing, nil
		}
		return nil, err
	}

	return s.repos.Users.GetByEmail(ctx, normalized)
}

func normalizedRole(role string) string {
	value := strings.TrimSpace(strings.ToLower(role))
	switch value {
	case "admin", "editor", "viewer":
		return value
	default:
		return "viewer"
	}
}

func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}
