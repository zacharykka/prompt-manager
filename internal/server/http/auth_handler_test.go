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
	"github.com/zacharykka/prompt-manager/internal/config"
	"github.com/zacharykka/prompt-manager/internal/infra/database"
	"github.com/zacharykka/prompt-manager/internal/infra/repository"
	"github.com/zacharykka/prompt-manager/internal/service/auth"
)

func setupAuthHandler(t *testing.T) (*AuthHandler, func()) {
	t.Helper()
	dsn := "file:auth_handler_test.db?mode=memory&cache=shared&_fk=1"
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
	svc := auth.NewService(repos, config.AuthConfig{
		AccessTokenSecret:  "abcdefghijklmnopqrstuvwxyz123456",
		RefreshTokenSecret: "abcdefghijklmnopqrstuvwxyz1234567890",
		AccessTokenTTL:     15 * 60 * 1e9,
		RefreshTokenTTL:    24 * 60 * 60 * 1e9,
		APIKeyHashSecret:   "abcdefghijklmnopqrstuvwxyz098765",
	})
	handler := NewAuthHandler(svc)

	cleanup := func() { _ = db.Close() }
	return handler, cleanup
}

func TestAuthHandler_RegisterAndLogin(t *testing.T) {
	handler, cleanup := setupAuthHandler(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/auth"))

	registerPayload := map[string]string{
		"email":    "user@example.com",
		"password": "password123",
		"role":     "admin",
	}
	registerBody, _ := json.Marshal(registerPayload)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("register failed: %d %s", rec.Code, rec.Body.String())
	}

	loginPayload := map[string]string{
		"email":    "user@example.com",
		"password": "password123",
	}
	loginBody, _ := json.Marshal(loginPayload)
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", loginRec.Code, loginRec.Body.String())
	}
}

func TestAuthHandler_RegisterInvalidRole(t *testing.T) {
	handler, cleanup := setupAuthHandler(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/auth"))

	payload := map[string]string{
		"email":    "user2@example.com",
		"password": "password123",
		"role":     "invalid",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", rec.Code)
	}
}

func TestAuthHandler_LoginWrongPassword(t *testing.T) {
	handler, cleanup := setupAuthHandler(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/auth"))

	// register first
	registerPayload := map[string]string{
		"email":    "user3@example.com",
		"password": "password123",
	}
	registerBody, _ := json.Marshal(registerPayload)
	registerReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	router.ServeHTTP(registerRec, registerReq)
	if registerRec.Code != http.StatusOK {
		t.Fatalf("register failed: %d %s", registerRec.Code, registerRec.Body.String())
	}

	// login with wrong password
	loginPayload := map[string]string{
		"email":    "user3@example.com",
		"password": "badpass",
	}
	loginBody, _ := json.Marshal(loginPayload)
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusBadRequest && loginRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 400/401 got %d", loginRec.Code)
	}
}

func TestAuthHandler_RefreshInvalidToken(t *testing.T) {
	handler, cleanup := setupAuthHandler(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/auth"))

	payload := map[string]string{"refresh_token": "invalid"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rec.Code)
	}
}
