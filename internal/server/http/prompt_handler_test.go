package http

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zacharykka/prompt-manager/internal/infra/database"
	"github.com/zacharykka/prompt-manager/internal/infra/repository"
	"github.com/zacharykka/prompt-manager/internal/middleware"
	promptsvc "github.com/zacharykka/prompt-manager/internal/service/prompt"
)

func setupPromptHandler(t *testing.T) (*PromptHandler, func()) {
	t.Helper()
	dsn := "file:prompt_handler_test.db?mode=memory&cache=shared&_fk=1"
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
	service := promptsvc.NewService(repos)
	handler := NewPromptHandler(service)

	cleanup := func() { _ = db.Close() }
	return handler, cleanup
}

func TestPromptHandler_CreateAndList(t *testing.T) {
	handler, cleanup := setupPromptHandler(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set(middleware.UserContextKey, "tester")
		ctx.Set(middleware.UserRoleContextKey, middleware.RoleAdmin)
		ctx.Next()
	})
	handler.RegisterRoutes(router.Group("/prompts"))

	payload := map[string]interface{}{
		"name": "Greeting",
		"tags": []string{"demo"},
		"body": "Hello there",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/prompts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d, body=%s", rec.Code, rec.Body.String())
	}

	// list prompts
	listReq := httptest.NewRequest(http.MethodGet, "/prompts?search=Gree", nil)
	listRec := httptest.NewRecorder()

	router.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", listRec.Code)
	}

	var listResp struct {
		Data struct {
			Items []struct {
				Name string  `json:"name"`
				Body *string `json:"active_version_body"`
			} `json:"items"`
			Meta struct {
				Total   int  `json:"total"`
				Limit   int  `json:"limit"`
				Offset  int  `json:"offset"`
				HasMore bool `json:"hasMore"`
			} `json:"meta"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	if listResp.Data.Meta.Total != 1 {
		t.Fatalf("expected total 1 got %d", listResp.Data.Meta.Total)
	}
	if len(listResp.Data.Items) != 1 || listResp.Data.Items[0].Name != "Greeting" {
		t.Fatalf("unexpected list response: %s", listRec.Body.String())
	}
	if listResp.Data.Items[0].Body == nil || *listResp.Data.Items[0].Body != "Hello there" {
		t.Fatalf("expected active version body, got %s", listRec.Body.String())
	}
}

func TestPromptHandler_CreateVersion(t *testing.T) {
	handler, cleanup := setupPromptHandler(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set(middleware.UserContextKey, "tester")
		ctx.Set(middleware.UserRoleContextKey, middleware.RoleAdmin)
		ctx.Next()
	})
	handler.RegisterRoutes(router.Group("/prompts"))

	// create prompt first
	createPayload := map[string]interface{}{"name": "Welcome"}
	createBody, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, "/prompts", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create prompt failed: status=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Prompt struct {
				ID string `json:"id"`
			} `json:"prompt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.Prompt.ID == "" {
		t.Fatalf("expected prompt id in response body=%s", rec.Body.String())
	}

	versionPayload := map[string]interface{}{
		"body":     "Hello",
		"activate": true,
	}
	versionBody, _ := json.Marshal(versionPayload)
	versionReq := httptest.NewRequest(http.MethodPost, "/prompts/"+resp.Data.Prompt.ID+"/versions", bytes.NewReader(versionBody))
	versionReq.Header.Set("Content-Type", "application/json")
	versionRec := httptest.NewRecorder()

	router.ServeHTTP(versionRec, versionReq)

	if versionRec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", versionRec.Code, versionRec.Body.String())
	}
}

func TestPromptHandler_GetStats(t *testing.T) {
	handler, cleanup := setupPromptHandler(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set(middleware.UserContextKey, "tester")
		ctx.Set(middleware.UserRoleContextKey, middleware.RoleAdmin)
		ctx.Next()
	})
	handler.RegisterRoutes(router.Group("/prompts"))

	// create prompt
	createPayload := map[string]interface{}{"name": "Stats"}
	createBody, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, "/prompts", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create prompt failed: %d %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Prompt struct {
				ID string `json:"id"`
			} `json:"prompt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	statsReq := httptest.NewRequest(http.MethodGet, "/prompts/"+resp.Data.Prompt.ID+"/stats", nil)
	statsRec := httptest.NewRecorder()
	router.ServeHTTP(statsRec, statsReq)
	if statsRec.Code != http.StatusOK {
		t.Fatalf("stats failed: %d %s", statsRec.Code, statsRec.Body.String())
	}
}
