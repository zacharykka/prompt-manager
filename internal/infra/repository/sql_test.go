package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zacharykka/prompt-manager/internal/domain"
	"github.com/zacharykka/prompt-manager/internal/infra/database"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	dsn := "file:repo_test.db?mode=memory&cache=shared&_fk=1"
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

	cleanup := func() {
		_ = db.Close()
	}
	return db, cleanup
}

func TestUserRepository_CreateAndGet(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewSQLRepositories(db, database.NewDialect("sqlite"))

	ctx := context.Background()
	userID := uuid.NewString()

	user := &domain.User{ID: userID, Email: "user@example.com", HashedPassword: "hashed", Role: "admin"}
	if err := repos.Users.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	stored, err := repos.Users.GetByEmail(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if stored.ID != userID {
		t.Fatalf("expected id %s got %s", userID, stored.ID)
	}

	if err := repos.Users.UpdateLastLogin(ctx, userID); err != nil {
		t.Fatalf("update last login: %v", err)
	}

	updated, err := repos.Users.GetByEmail(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if updated.LastLoginAt == nil {
		t.Fatalf("expected last login timestamp")
	}
	if time.Since(*updated.LastLoginAt) > time.Minute {
		t.Fatalf("last login timestamp too old")
	}
}

func TestPromptRepositories_Workflow(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewSQLRepositories(db, database.NewDialect("sqlite"))
	ctx := context.Background()

	userID := uuid.NewString()
	if err := repos.Users.Create(ctx, &domain.User{ID: userID, Email: "admin@example.com", HashedPassword: "hashed", Role: "admin"}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	promptID := uuid.NewString()
	tags := json.RawMessage(`["tag1","tag2"]`)
	prompt := &domain.Prompt{
		ID:        promptID,
		Name:      "Prompt-A",
		Tags:      tags,
		CreatedBy: &userID,
	}
	if err := repos.Prompts.Create(ctx, prompt); err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	storedPrompt, err := repos.Prompts.GetByID(ctx, promptID)
	if err != nil {
		t.Fatalf("get prompt: %v", err)
	}
	if string(storedPrompt.Tags) != string(tags) {
		t.Fatalf("unexpected tags: %s", storedPrompt.Tags)
	}

	byName, err := repos.Prompts.GetByName(ctx, "Prompt-A", false)
	if err != nil {
		t.Fatalf("get by name active: %v", err)
	}
	if byName.ID != promptID {
		t.Fatalf("expected id %s got %s", promptID, byName.ID)
	}
	if byName.Status != "active" {
		t.Fatalf("expected status active got %s", byName.Status)
	}

	versionID := uuid.NewString()
	schema := json.RawMessage(`{"vars":[{"name":"city"}]}`)
	version := &domain.PromptVersion{
		ID:              versionID,
		PromptID:        promptID,
		VersionNumber:   1,
		Body:            "Hello {{.city}}",
		VariablesSchema: schema,
		Status:          "published",
		CreatedBy:       &userID,
	}
	if err := repos.PromptVersions.Create(ctx, version); err != nil {
		t.Fatalf("create version: %v", err)
	}

	latest, err := repos.PromptVersions.GetLatestVersionNumber(ctx, promptID)
	if err != nil {
		t.Fatalf("latest number: %v", err)
	}
	if latest != 1 {
		t.Fatalf("expected latest version 1 got %d", latest)
	}

	body := "Hello {{.city}}"
	if err := repos.Prompts.UpdateActiveVersion(ctx, promptID, &versionID, &body); err != nil {
		t.Fatalf("update active version: %v", err)
	}

	updatedPrompt, err := repos.Prompts.GetByID(ctx, promptID)
	if err != nil {
		t.Fatalf("get prompt: %v", err)
	}
	if updatedPrompt.ActiveVersionID == nil || *updatedPrompt.ActiveVersionID != versionID {
		t.Fatalf("expected active version %s got %v", versionID, updatedPrompt.ActiveVersionID)
	}
	if updatedPrompt.Body == nil || *updatedPrompt.Body != "Hello {{.city}}" {
		t.Fatalf("expected prompt body, got %v", updatedPrompt.Body)
	}

	execLog := &domain.PromptExecutionLog{
		ID:               uuid.NewString(),
		PromptID:         promptID,
		PromptVersionID:  versionID,
		UserID:           &userID,
		Status:           "success",
		DurationMs:       120,
		RequestPayload:   json.RawMessage(`{"input":"Hello"}`),
		ResponseMetadata: json.RawMessage(`{"output":"World"}`),
	}
	if err := repos.PromptExecutionLog.Create(ctx, execLog); err != nil {
		t.Fatalf("create exec log: %v", err)
	}

	logs, err := repos.PromptExecutionLog.ListRecent(ctx, promptID, 10)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log got %d", len(logs))
	}
	if logs[0].DurationMs != 120 {
		t.Fatalf("unexpected duration: %d", logs[0].DurationMs)
	}

	stats, err := repos.PromptExecutionLog.AggregateUsage(ctx, promptID, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("aggregate usage: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat entry got %d", len(stats))
	}
	if stats[0].TotalCalls != 1 || stats[0].SuccessCalls != 1 {
		t.Fatalf("unexpected stats %+v", stats[0])
	}

	if err := repos.Prompts.Delete(ctx, promptID); err != nil {
		t.Fatalf("soft delete prompt: %v", err)
	}
	if _, err := repos.Prompts.GetByID(ctx, promptID); err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete got %v", err)
	}
	deletedByName, err := repos.Prompts.GetByName(ctx, "Prompt-A", true)
	if err != nil {
		t.Fatalf("get by name include deleted: %v", err)
	}
	if deletedByName.Status != "deleted" {
		t.Fatalf("expected deleted status got %s", deletedByName.Status)
	}
	if deletedByName.DeletedAt == nil {
		t.Fatalf("expected deleted_at to be set")
	}

	if err := repos.Prompts.Restore(ctx, promptID, domain.PromptRestoreParams{}); err != nil {
		t.Fatalf("restore prompt: %v", err)
	}
	restoredAfter, err := repos.Prompts.GetByID(ctx, promptID)
	if err != nil {
		t.Fatalf("get restored prompt: %v", err)
	}
	if restoredAfter.Status != "active" {
		t.Fatalf("expected active status got %s", restoredAfter.Status)
	}
	if restoredAfter.DeletedAt != nil {
		t.Fatalf("expected deleted_at cleared")
	}
	if err := repos.Prompts.Delete(ctx, promptID); err != nil {
		t.Fatalf("delete after restore: %v", err)
	}
	if _, err := repos.Prompts.GetByID(ctx, promptID); err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound after second delete got %v", err)
	}

	if err := repos.Prompts.Delete(ctx, promptID); err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound on repeated delete got %v", err)
	}
	listed, err := repos.Prompts.List(ctx, domain.PromptListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("list prompts after delete: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("expected no prompts after delete got %d", len(listed))
	}
}
