package auth

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zacharykka/prompt-manager/internal/config"
	domain "github.com/zacharykka/prompt-manager/internal/domain"
	"github.com/zacharykka/prompt-manager/internal/infra/database"
	"github.com/zacharykka/prompt-manager/internal/infra/repository"
)

func setupAuthTestService(t *testing.T) (*Service, func()) {
	t.Helper()
	dsn := "file:auth_service_test.db?mode=memory&cache=shared&_fk=1"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	migrationPath := filepath.Join("..", "..", "..", "db", "migrations", "000001_init.up.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := db.Exec(string(migrationSQL)); err != nil {
		t.Fatalf("exec migration: %v", err)
	}

	repos := repository.NewSQLRepositories(db, database.NewDialect("sqlite"))

	svc := NewService(repos, config.AuthConfig{
		AccessTokenSecret:  "access-secret",
		RefreshTokenSecret: "refresh-secret",
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    24 * time.Hour,
	})

	cleanup := func() {
		_ = db.Close()
	}
	return svc, cleanup
}

func createTenant(t *testing.T, svc *Service, tenantID string) {
	t.Helper()
	if err := svc.repos.Tenants.Create(context.Background(), &domain.Tenant{ID: tenantID, Name: "Tenant"}); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
}

func TestRegisterAndLogin(t *testing.T) {
	svc, cleanup := setupAuthTestService(t)
	defer cleanup()

	tenantID := uuid.NewString()
	createTenant(t, svc, tenantID)

	user, err := svc.Register(context.Background(), tenantID, "user@example.com", "password123", "admin")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.Role != "admin" {
		t.Fatalf("expected role admin got %s", user.Role)
	}

	tokens, loggedInUser, err := svc.Login(context.Background(), tenantID, "user@example.com", "password123")
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

	tenantID := uuid.NewString()
	createTenant(t, svc, tenantID)

	if _, err := svc.Register(context.Background(), tenantID, "user@example.com", "password123", ""); err != nil {
		t.Fatalf("register: %v", err)
	}

	if _, _, err := svc.Login(context.Background(), tenantID, "user@example.com", "wrong"); err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials got %v", err)
	}
}

func TestRefresh(t *testing.T) {
	svc, cleanup := setupAuthTestService(t)
	defer cleanup()

	tenantID := uuid.NewString()
	createTenant(t, svc, tenantID)

	_, err := svc.Register(context.Background(), tenantID, "user@example.com", "password123", "")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	tokens, _, err := svc.Login(context.Background(), tenantID, "user@example.com", "password123")
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
