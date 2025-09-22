package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/zacharykka/prompt-manager/internal/config"
	domain "github.com/zacharykka/prompt-manager/internal/domain"
	authutil "github.com/zacharykka/prompt-manager/pkg/auth"
)

// Service 封装认证逻辑。
type Service struct {
	repos *domain.Repositories
	cfg   config.AuthConfig
	nowFn func() time.Time
}

// Tokens 表示访问令牌与刷新令牌。
type Tokens struct {
	AccessToken           string    `json:"access_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshToken          string    `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
}

// NewService 创建认证服务。
func NewService(repos *domain.Repositories, cfg config.AuthConfig) *Service {
	return &Service{
		repos: repos,
		cfg:   cfg,
		nowFn: time.Now,
	}
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
