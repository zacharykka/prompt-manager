package prompt

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	domain "github.com/zacharykka/prompt-manager/internal/domain"
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
	migration2Path := filepath.Join("..", "..", "..", "db", "migrations", "000002_add_prompt_body.up.sql")
	migration2SQL, err := os.ReadFile(migration2Path)
	if err != nil {
		t.Fatalf("read migration 2: %v", err)
	}
	if _, err := db.Exec(string(migration2SQL)); err != nil {
		t.Fatalf("exec migration 2: %v", err)
	}
	migration3Path := filepath.Join("..", "..", "..", "db", "migrations", "000003_prompt_soft_delete.up.sql")
	migration3SQL, err := os.ReadFile(migration3Path)
	if err != nil {
		t.Fatalf("read migration 3: %v", err)
	}
	if _, err := db.Exec(string(migration3SQL)); err != nil {
		t.Fatalf("exec migration 3: %v", err)
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
	if updated.Body == nil || *updated.Body != "Hello, {{.name}}!" {
		t.Fatalf("unexpected prompt body: %v", updated.Body)
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

func TestGetExecutionStats(t *testing.T) {
	svc, cleanup := setupPromptService(t)
	defer cleanup()

	prompt, err := svc.CreatePrompt(context.Background(), CreatePromptInput{Name: "Stats"})
	if err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	// create versions and logs via repository since service does not expose logging yet
	version, err := svc.CreatePromptVersion(context.Background(), CreatePromptVersionInput{
		PromptID: prompt.ID,
		Body:     "test",
		Status:   "published",
		Activate: true,
	})
	if err != nil {
		t.Fatalf("create version: %v", err)
	}

	repos := svc.repos
	for i := 0; i < 3; i++ {
		status := "success"
		if i == 2 {
			status = "failed"
		}
		if err := repos.PromptExecutionLog.Create(context.Background(), &domain.PromptExecutionLog{
			ID:              uuid.NewString(),
			PromptID:        prompt.ID,
			PromptVersionID: version.ID,
			Status:          status,
			DurationMs:      int64(100 + i*10),
		}); err != nil {
			t.Fatalf("create log: %v", err)
		}
	}

	stats, err := svc.GetExecutionStats(context.Background(), prompt.ID, 7)
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	if len(stats) == 0 {
		t.Fatalf("expected stats entry")
	}
	if stats[0].TotalCalls != 3 {
		t.Fatalf("unexpected total calls: %d", stats[0].TotalCalls)
	}
}

func TestListPromptsWithSearch(t *testing.T) {
	svc, cleanup := setupPromptService(t)
	defer cleanup()

	if _, err := svc.CreatePrompt(context.Background(), CreatePromptInput{Name: "Alpha greeting"}); err != nil {
		t.Fatalf("create alpha: %v", err)
	}
	if _, err := svc.CreatePrompt(context.Background(), CreatePromptInput{Name: "Beta message"}); err != nil {
		t.Fatalf("create beta: %v", err)
	}

	prompts, total, err := svc.ListPrompts(context.Background(), ListPromptsOptions{
		Limit:  1,
		Search: "a",
	})
	if err != nil {
		t.Fatalf("list prompts: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2 got %d", total)
	}
	if len(prompts) != 1 {
		t.Fatalf("expected page size 1 got %d", len(prompts))
	}

	secondPage, _, err := svc.ListPrompts(context.Background(), ListPromptsOptions{
		Limit:  1,
		Offset: 1,
		Search: "a",
	})
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if len(secondPage) != 1 {
		t.Fatalf("expected second page 1 item got %d", len(secondPage))
	}
}

func TestUpdatePrompt(t *testing.T) {
	svc, cleanup := setupPromptService(t)
	defer cleanup()

	ctx := context.Background()
	prompt, err := svc.CreatePrompt(ctx, CreatePromptInput{Name: "Original"})
	if err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	second, err := svc.CreatePrompt(ctx, CreatePromptInput{Name: "Second"})
	if err != nil {
		t.Fatalf("create prompt second: %v", err)
	}

	newName := "Updated Name"
	desc := "  Detail  "
	tags := []string{"alpha", "beta"}
	updated, err := svc.UpdatePrompt(ctx, UpdatePromptInput{
		PromptID:    prompt.ID,
		Name:        &newName,
		Description: &desc,
		Tags:        &tags,
	})
	if err != nil {
		t.Fatalf("update prompt: %v", err)
	}
	if updated.Name != newName {
		t.Fatalf("expected name %s got %s", newName, updated.Name)
	}
	if updated.Description == nil || *updated.Description != "Detail" {
		t.Fatalf("expected trimmed description got %v", updated.Description)
	}
	var tagsResult []string
	if err := json.Unmarshal(updated.Tags, &tagsResult); err != nil {
		t.Fatalf("unmarshal tags: %v", err)
	}
	if len(tagsResult) != 2 || tagsResult[0] != "alpha" || tagsResult[1] != "beta" {
		t.Fatalf("unexpected tags payload: %v", tagsResult)
	}

	// 清空描述与标签
	emptyDesc := "  "
	var nilTags []string
	cleared, err := svc.UpdatePrompt(ctx, UpdatePromptInput{
		PromptID:    prompt.ID,
		Description: &emptyDesc,
		Tags:        &nilTags,
	})
	if err != nil {
		t.Fatalf("clear prompt fields: %v", err)
	}
	if cleared.Description != nil {
		t.Fatalf("expected description cleared got %v", cleared.Description)
	}
	if len(cleared.Tags) != 0 {
		t.Fatalf("expected tags cleared got %s", string(cleared.Tags))
	}

	// 重名校验
	if _, err := svc.UpdatePrompt(ctx, UpdatePromptInput{PromptID: second.ID, Name: &newName}); err != ErrPromptAlreadyExists {
		t.Fatalf("expected ErrPromptAlreadyExists got %v", err)
	}

	// 无字段更新
	if _, err := svc.UpdatePrompt(ctx, UpdatePromptInput{PromptID: prompt.ID}); err != ErrNoFieldsToUpdate {
		t.Fatalf("expected ErrNoFieldsToUpdate got %v", err)
	}
}

func TestDeletePrompt(t *testing.T) {
	svc, cleanup := setupPromptService(t)
	defer cleanup()

	ctx := context.Background()
	prompt, err := svc.CreatePrompt(ctx, CreatePromptInput{Name: "ToDelete"})
	if err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	if err := svc.DeletePrompt(ctx, prompt.ID, "tester@example.com"); err != nil {
		t.Fatalf("delete prompt: %v", err)
	}

	logs, err := svc.repos.PromptAuditLog.ListByPrompt(ctx, prompt.ID, 10)
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log got %d", len(logs))
	}
	if logs[0].Action != "prompt.deleted" {
		t.Fatalf("unexpected audit action %s", logs[0].Action)
	}
	if logs[0].CreatedBy == nil || *logs[0].CreatedBy != "tester@example.com" {
		t.Fatalf("expected audit actor tester@example.com got %v", logs[0].CreatedBy)
	}

	if _, err := svc.GetPrompt(ctx, prompt.ID); err != ErrPromptNotFound {
		t.Fatalf("expected ErrPromptNotFound got %v", err)
	}

	if err := svc.DeletePrompt(ctx, prompt.ID, "tester@example.com"); err != ErrPromptNotFound {
		t.Fatalf("expected ErrPromptNotFound on second delete got %v", err)
	}
}
