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

	cleanup := func() {
		_ = db.Close()
	}
	return db, cleanup
}

func TestTenantRepository_CreateAndGet(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewSQLRepositories(db, database.NewDialect("sqlite"))

	ctx := context.Background()
	tenantID := uuid.NewString()
	name := "Acme"
	desc := "Test Tenant"

	tenant := &domain.Tenant{ID: tenantID, Name: name, Description: &desc}
	if err := repos.Tenants.Create(ctx, tenant); err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	stored, err := repos.Tenants.GetByID(ctx, tenantID)
	if err != nil {
		t.Fatalf("get tenant: %v", err)
	}
	if stored.Name != name {
		t.Fatalf("expected name %s got %s", name, stored.Name)
	}
	if stored.Description == nil || *stored.Description != desc {
		t.Fatalf("expected description %s got %v", desc, stored.Description)
	}

	list, err := repos.Tenants.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("list tenants: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 tenant got %d", len(list))
	}
}

func TestPromptRepositories_Workflow(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewSQLRepositories(db, database.NewDialect("sqlite"))
	ctx := context.Background()

	tenantID := uuid.NewString()
	tenant := &domain.Tenant{ID: tenantID, Name: "Tenant"}
	if err := repos.Tenants.Create(ctx, tenant); err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	userID := uuid.NewString()
	user := &domain.User{
		ID:             userID,
		TenantID:       tenantID,
		Email:          "user@example.com",
		HashedPassword: "hashed",
		Role:           "admin",
	}
	if err := repos.Users.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	promptID := uuid.NewString()
	tags := json.RawMessage(`["tag1","tag2"]`)
	prompt := &domain.Prompt{
		ID:        promptID,
		TenantID:  tenantID,
		Name:      "Prompt-A",
		Tags:      tags,
		CreatedBy: &userID,
	}
	if err := repos.Prompts.Create(ctx, prompt); err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	storedPrompt, err := repos.Prompts.GetByID(ctx, tenantID, promptID)
	if err != nil {
		t.Fatalf("get prompt: %v", err)
	}
	if string(storedPrompt.Tags) != string(tags) {
		t.Fatalf("unexpected tags: %s", storedPrompt.Tags)
	}

	versionID := uuid.NewString()
	schema := json.RawMessage(`{"vars":[{"name":"city"}]}`)
	version := &domain.PromptVersion{
		ID:              versionID,
		TenantID:        tenantID,
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

	latest, err := repos.PromptVersions.GetLatestVersionNumber(ctx, tenantID, promptID)
	if err != nil {
		t.Fatalf("latest number: %v", err)
	}
	if latest != 1 {
		t.Fatalf("expected latest version 1 got %d", latest)
	}

	if err := repos.Prompts.UpdateActiveVersion(ctx, tenantID, promptID, &versionID); err != nil {
		t.Fatalf("update active version: %v", err)
	}

	updatedPrompt, err := repos.Prompts.GetByID(ctx, tenantID, promptID)
	if err != nil {
		t.Fatalf("get prompt: %v", err)
	}
	if updatedPrompt.ActiveVersionID == nil || *updatedPrompt.ActiveVersionID != versionID {
		t.Fatalf("expected active version %s got %v", versionID, updatedPrompt.ActiveVersionID)
	}

	execLog := &domain.PromptExecutionLog{
		ID:               uuid.NewString(),
		TenantID:         tenantID,
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

	logs, err := repos.PromptExecutionLog.ListRecent(ctx, tenantID, promptID, 10)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log got %d", len(logs))
	}
	if logs[0].DurationMs != 120 {
		t.Fatalf("unexpected duration: %d", logs[0].DurationMs)
	}
}

func TestUserRepository_UpdateLastLogin(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewSQLRepositories(db, database.NewDialect("sqlite"))
	ctx := context.Background()

	tenantID := uuid.NewString()
	if err := repos.Tenants.Create(ctx, &domain.Tenant{ID: tenantID, Name: "Tenant"}); err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	userID := uuid.NewString()
	user := &domain.User{
		ID:             userID,
		TenantID:       tenantID,
		Email:          "user@example.com",
		HashedPassword: "hashed",
	}
	if err := repos.Users.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	if err := repos.Users.UpdateLastLogin(ctx, tenantID, userID); err != nil {
		t.Fatalf("update last login: %v", err)
	}

	stored, err := repos.Users.GetByEmail(ctx, tenantID, "user@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if stored.LastLoginAt == nil {
		t.Fatalf("expected last login timestamp")
	}
	if time.Since(*stored.LastLoginAt) > time.Minute {
		t.Fatalf("last login timestamp too old: %v", stored.LastLoginAt)
	}
}
