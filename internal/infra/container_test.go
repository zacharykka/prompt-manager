package infra

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/zacharykka/prompt-manager/internal/config"
	domain "github.com/zacharykka/prompt-manager/internal/domain"
	"github.com/zacharykka/prompt-manager/internal/infra/database"
	"github.com/zacharykka/prompt-manager/internal/infra/repository"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

func prepareTestRepo(t *testing.T) (*domain.Repositories, func()) {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "app.db") + "?_fk=1"

	schemaPath := filepath.Join("..", "..", "db", "migrations", "000001_init.up.sql")
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("exec migration: %v", err)
	}

	repos := repository.NewSQLRepositories(db, database.NewDialect("sqlite"))
	cleanup := func() { _ = db.Close() }
	return repos, cleanup
}

func TestEnsureDefaultAdminCreatesUser(t *testing.T) {
	repos, cleanup := prepareTestRepo(t)
	defer cleanup()

	t.Setenv("PROMPT_MANAGER_INIT_ADMIN_EMAIL", "seed@example.com")
	t.Setenv("PROMPT_MANAGER_INIT_ADMIN_PASSWORD", "super-secure-password-1234567890")
	t.Setenv("PROMPT_MANAGER_INIT_ADMIN_ROLE", "editor")

	cfg := &config.Config{}
	if err := ensureDefaultAdmin(context.Background(), cfg, repos, zap.NewNop()); err != nil {
		t.Fatalf("ensureDefaultAdmin failed: %v", err)
	}

	user, err := repos.Users.GetByEmail(context.Background(), "seed@example.com")
	if err != nil {
		t.Fatalf("expected seeded user: %v", err)
	}
	if user.Role != "editor" {
		t.Fatalf("expected role editor got %s", user.Role)
	}
}

func TestEnsureDefaultAdminIdempotent(t *testing.T) {
	repos, cleanup := prepareTestRepo(t)
	defer cleanup()

	t.Setenv("PROMPT_MANAGER_INIT_ADMIN_EMAIL", "seed@example.com")
	t.Setenv("PROMPT_MANAGER_INIT_ADMIN_PASSWORD", "super-secure-password-1234567890")

	cfg := &config.Config{}
	if err := ensureDefaultAdmin(context.Background(), cfg, repos, zap.NewNop()); err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if err := ensureDefaultAdmin(context.Background(), cfg, repos, zap.NewNop()); err != nil {
		t.Fatalf("second call failed: %v", err)
	}
}

func TestEnsureDefaultAdminSkippedWhenEnvMissing(t *testing.T) {
	repos, cleanup := prepareTestRepo(t)
	defer cleanup()

	os.Unsetenv("PROMPT_MANAGER_INIT_ADMIN_EMAIL")
	os.Unsetenv("PROMPT_MANAGER_INIT_ADMIN_PASSWORD")

	cfg := &config.Config{}
	if err := ensureDefaultAdmin(context.Background(), cfg, repos, zap.NewNop()); err != nil {
		t.Fatalf("should succeed even when env missing: %v", err)
	}
	if _, err := repos.Users.GetByEmail(context.Background(), "seed@example.com"); err == nil {
		t.Fatalf("unexpected user created without env")
	}
}

func TestEnsureDefaultAdminUsesConfig(t *testing.T) {
	repos, cleanup := prepareTestRepo(t)
	defer cleanup()

	cfg := &config.Config{
		Seed: config.SeedConfig{
			Admin: config.SeedAdminConfig{
				Email:    "config@example.com",
				Password: "super-secure-password-1234567890",
				Role:     "editor",
			},
		},
	}

	if err := ensureDefaultAdmin(context.Background(), cfg, repos, zap.NewNop()); err != nil {
		t.Fatalf("ensureDefaultAdmin with config failed: %v", err)
	}

	user, err := repos.Users.GetByEmail(context.Background(), "config@example.com")
	if err != nil {
		t.Fatalf("expected user created from config: %v", err)
	}
	if user.Role != "editor" {
		t.Fatalf("expected role editor got %s", user.Role)
	}
}
