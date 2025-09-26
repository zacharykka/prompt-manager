package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zacharykka/prompt-manager/internal/config"
	"github.com/zacharykka/prompt-manager/internal/infra/database"
	"github.com/zacharykka/prompt-manager/internal/infra/repository"
)

func setupAuthTestService(t *testing.T) (*Service, func()) {
	t.Helper()
	cfg := config.AuthConfig{
		AccessTokenSecret:  "access-secret",
		RefreshTokenSecret: "refresh-secret",
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    24 * time.Hour,
	}
	return setupAuthTestServiceWithConfig(t, cfg)
}

func setupAuthTestServiceWithConfig(t *testing.T, cfg config.AuthConfig, opts ...Option) (*Service, func()) {
	t.Helper()
	dsn := "file:auth_service_test.db?mode=memory&cache=shared&_fk=1"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	migrationDir := filepath.Join("..", "..", "..", "db", "migrations")
	migrationFiles := []string{
		"000001_init.up.sql",
		"000002_add_prompt_body.up.sql",
		"000003_prompt_soft_delete.up.sql",
		"000004_add_user_identities.up.sql",
	}
	for _, file := range migrationFiles {
		path := filepath.Join(migrationDir, file)
		migrationSQL, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", file, err)
		}
		if _, err := db.Exec(string(migrationSQL)); err != nil {
			t.Fatalf("exec migration %s: %v", file, err)
		}
	}

	repos := repository.NewSQLRepositories(db, database.NewDialect("sqlite"))

	svc := NewService(repos, cfg, opts...)

	cleanup := func() {
		_ = db.Close()
	}
	return svc, cleanup
}

func TestRegisterAndLogin(t *testing.T) {
	svc, cleanup := setupAuthTestService(t)
	defer cleanup()

	user, err := svc.Register(context.Background(), "user@example.com", "password123", "admin")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.Role != "admin" {
		t.Fatalf("expected role admin got %s", user.Role)
	}

	tokens, loggedInUser, err := svc.Login(context.Background(), "user@example.com", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("expected tokens to be generated")
	}
	if loggedInUser.ID != user.ID {
		t.Fatalf("expected same user")
	}
}

func TestLoginInvalidPassword(t *testing.T) {
	svc, cleanup := setupAuthTestService(t)
	defer cleanup()

	if _, err := svc.Register(context.Background(), "user@example.com", "password123", ""); err != nil {
		t.Fatalf("register: %v", err)
	}

	if _, _, err := svc.Login(context.Background(), "user@example.com", "wrong"); err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials got %v", err)
	}
}

func TestRefresh(t *testing.T) {
	svc, cleanup := setupAuthTestService(t)
	defer cleanup()

	if _, err := svc.Register(context.Background(), "user@example.com", "password123", ""); err != nil {
		t.Fatalf("register: %v", err)
	}

	tokens, _, err := svc.Login(context.Background(), "user@example.com", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	newTokens, _, err := svc.Refresh(context.Background(), tokens.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if newTokens.AccessToken == "" || newTokens.RefreshToken == "" {
		t.Fatalf("expected tokens to be generated")
	}
}

func TestGitHubAuthorizeURL(t *testing.T) {
	svc, cleanup := setupAuthTestServiceWithConfig(t, config.AuthConfig{
		AccessTokenSecret:  "access-secret",
		RefreshTokenSecret: "refresh-secret",
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    24 * time.Hour,
		GitHub: config.GitHubOAuthConfig{
			Enabled:      true,
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			RedirectURL:  "http://localhost:8080/api/v1/auth/github/callback",
			Scopes:       []string{"read:user", "user:email"},
			StateTTL:     time.Minute,
		},
	})
	defer cleanup()

	authorizeURL, err := svc.GitHubAuthorizeURL("https://app.example.com/finish", "web_message")
	if err != nil {
		t.Fatalf("GitHubAuthorizeURL error: %v", err)
	}

	parsed, err := url.Parse(authorizeURL)
	if err != nil {
		t.Fatalf("parse authorize url: %v", err)
	}

	query := parsed.Query()
	if got := query.Get("client_id"); got != "client-id" {
		t.Fatalf("unexpected client_id: %s", got)
	}
	if got := query.Get("redirect_uri"); got != "http://localhost:8080/api/v1/auth/github/callback" {
		t.Fatalf("unexpected redirect_uri: %s", got)
	}
	expectedScope := strings.Join([]string{"read:user", "user:email"}, " ")
	if got := query.Get("scope"); got != expectedScope {
		t.Fatalf("unexpected scope: %s", got)
	}

	state := query.Get("state")
	if state == "" {
		t.Fatalf("state should not be empty")
	}

	provider, redirectURI, mode, err := svc.parseOAuthState(state)
	if err != nil {
		t.Fatalf("parseOAuthState error: %v", err)
	}
	if provider != providerGitHub {
		t.Fatalf("expected provider %s got %s", providerGitHub, provider)
	}
	if redirectURI != "https://app.example.com/finish" {
		t.Fatalf("unexpected redirect uri: %s", redirectURI)
	}
	if mode != "web_message" {
		t.Fatalf("unexpected response mode: %s", mode)
	}
}

func TestHandleGitHubCallback_NewUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/oauth/access_token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"stub-token","token_type":"bearer","scope":"read:user user:email"}`))
		case "/user":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":12345,"login":"octocat","email":"","avatar_url":"https://avatars.example.com/u/12345"}`))
		case "/user/emails":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"email":"octocat@example.com","primary":true,"verified":true}]`))
		case "/user/orgs":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"login":"allowed-org"}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	httpClient := server.Client()
	httpClient.Timeout = 2 * time.Second

	cfg := config.AuthConfig{
		AccessTokenSecret:  "access-secret",
		RefreshTokenSecret: "refresh-secret",
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    24 * time.Hour,
		GitHub: config.GitHubOAuthConfig{
			Enabled:      true,
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			RedirectURL:  server.URL + "/callback",
			Scopes:       []string{"read:user", "user:email"},
			AllowedOrgs:  []string{"allowed-org"},
			StateTTL:     time.Minute,
		},
	}

	svc, cleanup := setupAuthTestServiceWithConfig(t, cfg, WithHTTPClient(httpClient), WithGitHubEndpoints(server.URL+"/authorize", server.URL+"/login/oauth/access_token", server.URL))
	defer cleanup()

	authorizeURL, err := svc.GitHubAuthorizeURL("", "")
	if err != nil {
		t.Fatalf("GitHubAuthorizeURL error: %v", err)
	}

	parsed, err := url.Parse(authorizeURL)
	if err != nil {
		t.Fatalf("parse authorize url: %v", err)
	}
	state := parsed.Query().Get("state")
	if state == "" {
		t.Fatalf("state should not be empty")
	}

	tokens, user, redirectURI, responseMode, err := svc.HandleGitHubCallback(context.Background(), "dummy-code", state)
	if err != nil {
		t.Fatalf("HandleGitHubCallback error: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("expected tokens to be populated")
	}
	if redirectURI != "" {
		t.Fatalf("unexpected redirect uri: %s", redirectURI)
	}
	if responseMode != "json" {
		t.Fatalf("unexpected response mode: %s", responseMode)
	}
	if user.Email != "octocat@example.com" {
		t.Fatalf("unexpected user email: %s", user.Email)
	}
	if user.Role != "viewer" {
		t.Fatalf("unexpected user role: %s", user.Role)
	}

	identity, err := svc.repos.UserIdentities.GetByProviderAndExternalID(context.Background(), providerGitHub, "12345")
	if err != nil {
		t.Fatalf("identity lookup error: %v", err)
	}
	if identity.UserID != user.ID {
		t.Fatalf("identity user mismatch")
	}
}

func TestHandleGitHubCallback_OrgRestriction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/oauth/access_token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"stub-token","token_type":"bearer"}`))
		case "/user":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":6789,"login":"user","email":"user@example.com"}`))
		case "/user/orgs":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"login":"other-org"}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	httpClient := server.Client()
	httpClient.Timeout = 2 * time.Second

	cfg := config.AuthConfig{
		AccessTokenSecret:  "access-secret",
		RefreshTokenSecret: "refresh-secret",
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    24 * time.Hour,
		GitHub: config.GitHubOAuthConfig{
			Enabled:      true,
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			RedirectURL:  server.URL + "/callback",
			AllowedOrgs:  []string{"allowed-org"},
			StateTTL:     time.Minute,
		},
	}

	svc, cleanup := setupAuthTestServiceWithConfig(t, cfg, WithHTTPClient(httpClient), WithGitHubEndpoints(server.URL+"/authorize", server.URL+"/login/oauth/access_token", server.URL))
	defer cleanup()

	authorizeURL, err := svc.GitHubAuthorizeURL("", "")
	if err != nil {
		t.Fatalf("GitHubAuthorizeURL error: %v", err)
	}

	parsed, err := url.Parse(authorizeURL)
	if err != nil {
		t.Fatalf("parse authorize url: %v", err)
	}
	state := parsed.Query().Get("state")
	if state == "" {
		t.Fatalf("state should not be empty")
	}

	_, _, _, _, err = svc.HandleGitHubCallback(context.Background(), "dummy-code", state)
	if !errors.Is(err, ErrOAuthOrgUnauthorized) {
		t.Fatalf("expected ErrOAuthOrgUnauthorized got %v", err)
	}
}
