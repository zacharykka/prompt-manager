package prompt

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/zacharykka/prompt-manager/internal/infra/database"
	"github.com/zacharykka/prompt-manager/internal/infra/repository"
)

func setupPromptService(t *testing.T) (*Service, func()) {
	t.Helper()
	dsn := "file:prompt_service_test.db?mode=memory&cache=shared&_fk=1"
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
	svc := NewService(repos)

	cleanup := func() { _ = db.Close() }
	return svc, cleanup
}

func TestCreatePromptAndVersion(t *testing.T) {
	svc, cleanup := setupPromptService(t)
	defer cleanup()

	prompt, err := svc.CreatePrompt(context.Background(), CreatePromptInput{
		Name:      "Welcome Message",
		Tags:      []string{"greeting"},
		CreatedBy: uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	version, err := svc.CreatePromptVersion(context.Background(), CreatePromptVersionInput{
		PromptID: prompt.ID,
		Body:     "Hello, {{.name}}!",
		VariablesSchema: map[string]interface{}{
			"vars": []map[string]string{{"name": "name"}},
		},
		Status:    "published",
		CreatedBy: uuid.NewString(),
		Activate:  true,
	})
	if err != nil {
		t.Fatalf("create prompt version: %v", err)
	}

	if version.VersionNumber != 1 {
		t.Fatalf("expected version number 1 got %d", version.VersionNumber)
	}

	updated, err := svc.GetPrompt(context.Background(), prompt.ID)
	if err != nil {
		t.Fatalf("get prompt: %v", err)
	}
	if updated.ActiveVersionID == nil || *updated.ActiveVersionID != version.ID {
		t.Fatalf("expected active version to be %s", version.ID)
	}

	versions, err := svc.ListPromptVersions(context.Background(), prompt.ID, 10, 0)
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("expected 1 version got %d", len(versions))
	}
}

func TestCreatePromptDuplicate(t *testing.T) {
	svc, cleanup := setupPromptService(t)
	defer cleanup()

	if _, err := svc.CreatePrompt(context.Background(), CreatePromptInput{Name: "Duplicate"}); err != nil {
		t.Fatalf("create prompt: %v", err)
	}
	if _, err := svc.CreatePrompt(context.Background(), CreatePromptInput{Name: "Duplicate"}); err != ErrPromptAlreadyExists {
		t.Fatalf("expected ErrPromptAlreadyExists got %v", err)
	}
}
