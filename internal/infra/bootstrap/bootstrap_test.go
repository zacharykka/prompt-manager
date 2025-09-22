package bootstrap

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
)

func setupRepo(t *testing.T) (*domain.Repositories, func()) {
	t.Helper()
	dsn := "file:bootstrap_test.db?mode=memory&cache=shared&_fk=1"
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
	cleanup := func() {
		_ = db.Close()
	}
	return repos, cleanup
}

func TestEnsureDefaultAdmin(t *testing.T) {
	repos, cleanup := setupRepo(t)
	defer cleanup()

	cfg := config.BootstrapConfig{
		Enabled:           true,
		TenantID:          "test-tenant",
		TenantName:        "Test Tenant",
		AdminEmail:        "admin",
		AdminPassword:     "password",
		AdminRole:         "admin",
		TenantDescription: "bootstrap test",
	}

	logger := zap.NewNop()

	if err := EnsureDefaultAdmin(context.Background(), repos, cfg, logger); err != nil {
		t.Fatalf("ensure default admin: %v", err)
	}

	// second call should be idempotent
	if err := EnsureDefaultAdmin(context.Background(), repos, cfg, logger); err != nil {
		t.Fatalf("ensure default admin second call: %v", err)
	}

	tenant, err := repos.Tenants.GetByID(context.Background(), cfg.TenantID)
	if err != nil {
		t.Fatalf("get tenant: %v", err)
	}
	if tenant.Name != cfg.TenantName {
		t.Fatalf("unexpected tenant name: %s", tenant.Name)
	}

	user, err := repos.Users.GetByEmail(context.Background(), cfg.TenantID, cfg.AdminEmail)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if user.Role != "admin" {
		t.Fatalf("unexpected role: %s", user.Role)
	}
}
