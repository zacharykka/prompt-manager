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
		ctx.Set(middleware.UserContextKey, "tester-id")
		ctx.Set(middleware.UserEmailContextKey, "tester@example.com")
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
				Name string `json:"name"`
				Body string `json:"body"`
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
	if listResp.Data.Items[0].Body != "Hello there" {
		t.Fatalf("expected active version body, got %s", listRec.Body.String())
	}
}

func TestPromptHandler_ListIncludesDeleted(t *testing.T) {
	handler, cleanup := setupPromptHandler(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set(middleware.UserContextKey, "tester-id")
		ctx.Set(middleware.UserEmailContextKey, "tester@example.com")
		ctx.Set(middleware.UserRoleContextKey, middleware.RoleAdmin)
		ctx.Next()
	})
	handler.RegisterRoutes(router.Group("/prompts"))

	createPayload := map[string]interface{}{"name": "Need recycle"}
	createBody, _ := json.Marshal(createPayload)
	createReq := httptest.NewRequest(http.MethodPost, "/prompts", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()

	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusOK {
		t.Fatalf("create prompt expected 200 got %d body=%s", createRec.Code, createRec.Body.String())
	}

	var createResp struct {
		Data struct {
			Prompt struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"prompt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}

	firstVersionPayload := map[string]interface{}{
		"body":             "Hello, world!",
		"variables_schema": map[string]interface{}{"foo": "bar"},
		"activate":         true,
	}
	firstBody, _ := json.Marshal(firstVersionPayload)
	firstReq := httptest.NewRequest(http.MethodPost, "/prompts/"+createResp.Data.Prompt.ID+"/versions", bytes.NewReader(firstBody))
	firstReq.Header.Set("Content-Type", "application/json")
	firstRec := httptest.NewRecorder()
	router.ServeHTTP(firstRec, firstReq)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("create first active version expected 200 got %d body=%s", firstRec.Code, firstRec.Body.String())
	}
	if createResp.Data.Prompt.ID == "" {
		t.Fatalf("expected prompt id in create response: %s", createRec.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/prompts/"+createResp.Data.Prompt.ID, nil)
	deleteRec := httptest.NewRecorder()

	router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete prompt expected 200 got %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}

	type listResponse struct {
		Data struct {
			Items []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"items"`
			Meta struct {
				Total int `json:"total"`
			} `json:"meta"`
		} `json:"data"`
	}

	activeReq := httptest.NewRequest(http.MethodGet, "/prompts", nil)
	activeRec := httptest.NewRecorder()
	router.ServeHTTP(activeRec, activeReq)

	if activeRec.Code != http.StatusOK {
		t.Fatalf("list active expected 200 got %d body=%s", activeRec.Code, activeRec.Body.String())
	}

	var activeResp listResponse
	if err := json.Unmarshal(activeRec.Body.Bytes(), &activeResp); err != nil {
		t.Fatalf("unmarshal active list: %v", err)
	}
	if activeResp.Data.Meta.Total != 0 {
		t.Fatalf("expected active total 0 got %d", activeResp.Data.Meta.Total)
	}
	if len(activeResp.Data.Items) != 0 {
		t.Fatalf("expected no active items got %d", len(activeResp.Data.Items))
	}

	deletedReq := httptest.NewRequest(http.MethodGet, "/prompts?includeDeleted=true", nil)
	deletedRec := httptest.NewRecorder()
	router.ServeHTTP(deletedRec, deletedReq)

	if deletedRec.Code != http.StatusOK {
		t.Fatalf("list deleted expected 200 got %d body=%s", deletedRec.Code, deletedRec.Body.String())
	}

	var deletedResp listResponse
	if err := json.Unmarshal(deletedRec.Body.Bytes(), &deletedResp); err != nil {
		t.Fatalf("unmarshal deleted list: %v", err)
	}
	if deletedResp.Data.Meta.Total == 0 {
		t.Fatalf("expected deleted total > 0 got %d", deletedResp.Data.Meta.Total)
	}
	if len(deletedResp.Data.Items) != 1 {
		t.Fatalf("expected one deleted item got %d", len(deletedResp.Data.Items))
	}
	if deletedResp.Data.Items[0].Status != "deleted" {
		t.Fatalf("expected item status deleted got %s", deletedResp.Data.Items[0].Status)
	}
}

func TestPromptHandler_Restore(t *testing.T) {
	handler, cleanup := setupPromptHandler(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set(middleware.UserContextKey, "tester-id")
		ctx.Set(middleware.UserEmailContextKey, "tester@example.com")
		ctx.Set(middleware.UserRoleContextKey, middleware.RoleAdmin)
		ctx.Next()
	})
	handler.RegisterRoutes(router.Group("/prompts"))

	createPayload := map[string]interface{}{"name": "Need restore"}
	createBody, _ := json.Marshal(createPayload)
	createReq := httptest.NewRequest(http.MethodPost, "/prompts", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()

	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusOK {
		t.Fatalf("create prompt expected 200 got %d body=%s", createRec.Code, createRec.Body.String())
	}

	var createResp struct {
		Data struct {
			Prompt struct {
				ID string `json:"id"`
			} `json:"prompt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/prompts/"+createResp.Data.Prompt.ID, nil)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete prompt expected 200 got %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}

	restoreReq := httptest.NewRequest(http.MethodPost, "/prompts/"+createResp.Data.Prompt.ID+"/restore", nil)
	restoreRec := httptest.NewRecorder()
	router.ServeHTTP(restoreRec, restoreReq)

	if restoreRec.Code != http.StatusOK {
		t.Fatalf("restore prompt expected 200 got %d body=%s", restoreRec.Code, restoreRec.Body.String())
	}

	var restoreResp struct {
		Data struct {
			Prompt struct {
				Status string `json:"status"`
			} `json:"prompt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(restoreRec.Body.Bytes(), &restoreResp); err != nil {
		t.Fatalf("unmarshal restore response: %v", err)
	}
	if restoreResp.Data.Prompt.Status != "active" {
		t.Fatalf("expected restored prompt status active got %s", restoreResp.Data.Prompt.Status)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/prompts", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list prompts expected 200 got %d body=%s", listRec.Code, listRec.Body.String())
	}

	var listResp struct {
		Data struct {
			Meta struct {
				Total int `json:"total"`
			} `json:"meta"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	if listResp.Data.Meta.Total != 1 {
		t.Fatalf("expected total 1 got %d", listResp.Data.Meta.Total)
	}
}

func TestPromptHandler_DiffVersion(t *testing.T) {
	handler, cleanup := setupPromptHandler(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set(middleware.UserContextKey, "tester-id")
		ctx.Set(middleware.UserEmailContextKey, "tester@example.com")
		ctx.Set(middleware.UserRoleContextKey, middleware.RoleAdmin)
		ctx.Next()
	})
	handler.RegisterRoutes(router.Group("/prompts"))

	createPayload := map[string]interface{}{
		"name": "Version Diff",
		"body": "Hello, world!",
	}
	createBody, _ := json.Marshal(createPayload)
	createReq := httptest.NewRequest(http.MethodPost, "/prompts", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create prompt expected 200 got %d body=%s", createRec.Code, createRec.Body.String())
	}

	var createResp struct {
		Data struct {
			Prompt struct {
				ID string `json:"id"`
			} `json:"prompt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}

	versionPayload := map[string]interface{}{
		"body":             "Hello, universe!",
		"variables_schema": map[string]interface{}{"foo": "baz", "new": 1},
	}
	versionBody, _ := json.Marshal(versionPayload)
	versionReq := httptest.NewRequest(http.MethodPost, "/prompts/"+createResp.Data.Prompt.ID+"/versions", bytes.NewReader(versionBody))
	versionReq.Header.Set("Content-Type", "application/json")
	versionRec := httptest.NewRecorder()
	router.ServeHTTP(versionRec, versionReq)
	if versionRec.Code != http.StatusOK {
		t.Fatalf("create version expected 200 got %d body=%s", versionRec.Code, versionRec.Body.String())
	}

	var versionResp struct {
		Data struct {
			Version struct {
				ID string `json:"id"`
			} `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(versionRec.Body.Bytes(), &versionResp); err != nil {
		t.Fatalf("unmarshal version response: %v", err)
	}

	diffReq := httptest.NewRequest(http.MethodGet, "/prompts/"+createResp.Data.Prompt.ID+"/versions/"+versionResp.Data.Version.ID+"/diff?compareTo=active", nil)
	diffRec := httptest.NewRecorder()
	router.ServeHTTP(diffRec, diffReq)
	if diffRec.Code != http.StatusOK {
		t.Fatalf("diff request expected 200 got %d body=%s", diffRec.Code, diffRec.Body.String())
	}

	var diffResp struct {
		Data struct {
			Diff struct {
				Body []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"body"`
				Variables *struct {
					Changes []struct {
						Key   string `json:"key"`
						Type  string `json:"type"`
						Left  string `json:"left"`
						Right string `json:"right"`
					} `json:"changes"`
				} `json:"variables_schema"`
			} `json:"diff"`
		} `json:"data"`
	}
	if err := json.Unmarshal(diffRec.Body.Bytes(), &diffResp); err != nil {
		t.Fatalf("unmarshal diff response: %v", err)
	}
	if len(diffResp.Data.Diff.Body) == 0 {
		t.Fatalf("expected diff body segments")
	}
	if diffResp.Data.Diff.Variables == nil || len(diffResp.Data.Diff.Variables.Changes) == 0 {
		t.Fatalf("expected variables diff changes")
	}
}

func TestPromptHandler_CreateVersion(t *testing.T) {
	handler, cleanup := setupPromptHandler(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set(middleware.UserContextKey, "tester-id")
		ctx.Set(middleware.UserEmailContextKey, "tester@example.com")
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

func TestPromptHandler_Update(t *testing.T) {
	handler, cleanup := setupPromptHandler(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set(middleware.UserContextKey, "tester-id")
		ctx.Set(middleware.UserEmailContextKey, "tester@example.com")
		ctx.Set(middleware.UserRoleContextKey, middleware.RoleAdmin)
		ctx.Next()
	})
	handler.RegisterRoutes(router.Group("/prompts"))

	createPayload := map[string]interface{}{"name": "Need Update"}
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
		t.Fatalf("unmarshal response: %v", err)
	}

	updatePayload := map[string]interface{}{
		"name":        "Updated",
		"description": " detail ",
		"tags":        []string{"x"},
	}
	updateBody, _ := json.Marshal(updatePayload)
	updateReq := httptest.NewRequest(http.MethodPatch, "/prompts/"+resp.Data.Prompt.ID, bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	router.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update prompt failed: %d %s", updateRec.Code, updateRec.Body.String())
	}

	var updateResp struct {
		Data struct {
			Prompt struct {
				Name        string          `json:"name"`
				Description *string         `json:"description"`
				Tags        json.RawMessage `json:"tags"`
			} `json:"prompt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updateResp); err != nil {
		t.Fatalf("unmarshal update: %v", err)
	}
	if updateResp.Data.Prompt.Name != "Updated" {
		t.Fatalf("expected updated name got %s", updateResp.Data.Prompt.Name)
	}
	if updateResp.Data.Prompt.Description == nil || *updateResp.Data.Prompt.Description != "detail" {
		t.Fatalf("expected trimmed description got %v", updateResp.Data.Prompt.Description)
	}
	var tagList []string
	if err := json.Unmarshal(updateResp.Data.Prompt.Tags, &tagList); err != nil || len(tagList) != 1 || tagList[0] != "x" {
		t.Fatalf("unexpected tags: %s", string(updateResp.Data.Prompt.Tags))
	}

	noChangeReq := httptest.NewRequest(http.MethodPatch, "/prompts/"+resp.Data.Prompt.ID, bytes.NewReader([]byte(`{}`)))
	noChangeReq.Header.Set("Content-Type", "application/json")
	noChangeRec := httptest.NewRecorder()
	router.ServeHTTP(noChangeRec, noChangeReq)
	if noChangeRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on empty update got %d", noChangeRec.Code)
	}
}

func TestPromptHandler_Delete(t *testing.T) {
	handler, cleanup := setupPromptHandler(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set(middleware.UserContextKey, "tester-id")
		ctx.Set(middleware.UserEmailContextKey, "tester@example.com")
		ctx.Set(middleware.UserRoleContextKey, middleware.RoleAdmin)
		ctx.Next()
	})
	handler.RegisterRoutes(router.Group("/prompts"))

	createPayload := map[string]interface{}{"name": "Delete Me"}
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
		t.Fatalf("unmarshal response: %v", err)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/prompts/"+resp.Data.Prompt.ID, nil)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete prompt failed: %d %s", deleteRec.Code, deleteRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/prompts/"+resp.Data.Prompt.ID, nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete got %d", getRec.Code)
	}
}

func TestPromptHandler_GetStats(t *testing.T) {
	handler, cleanup := setupPromptHandler(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set(middleware.UserContextKey, "tester-id")
		ctx.Set(middleware.UserEmailContextKey, "tester@example.com")
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
