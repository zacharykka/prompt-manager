package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zacharykka/prompt-manager/internal/config"
	"go.uber.org/zap"
)

func TestBuildCORSConfigExactOrigins(t *testing.T) {
	cfg := config.ServerConfig{}
	cfg.CORS.AllowOrigins = []string{"https://app.example.com"}

	corsCfg := buildCORSConfig(cfg)
	if corsCfg.AllowAllOrigins {
		t.Fatalf("expected not to allow all origins")
	}
	if len(corsCfg.AllowOrigins) != 1 || corsCfg.AllowOrigins[0] != "https://app.example.com" {
		t.Fatalf("expected exact allow origins, got %+v", corsCfg.AllowOrigins)
	}
}

func TestBuildCORSConfigAllowsWildcardPattern(t *testing.T) {
	cfg := config.ServerConfig{}
	cfg.CORS.AllowOrigins = []string{"https://*.example.com"}

	corsCfg := buildCORSConfig(cfg)
	if corsCfg.AllowOriginFunc == nil {
		t.Fatalf("expected AllowOriginFunc to be set for patterns")
	}
	if corsCfg.AllowOriginFunc("https://api.example.com") == false {
		t.Fatalf("expected wildcard pattern to match subdomain")
	}
	if corsCfg.AllowOriginFunc("https://example.org") {
		t.Fatalf("expected wildcard pattern not to match unrelated domain")
	}
}

func TestBuildCORSConfigAllowAll(t *testing.T) {
	cfg := config.ServerConfig{}
	cfg.CORS.AllowOrigins = []string{"*"}

	corsCfg := buildCORSConfig(cfg)
	if !corsCfg.AllowAllOrigins {
		t.Fatalf("expected AllowAllOrigins to be true when '*' configured")
	}
}

func TestSecurityHeadersIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		App: config.AppConfig{Name: "test", Env: "test"},
		Server: config.ServerConfig{
			CORS: config.CORSConfig{AllowOrigins: []string{"https://app.example.com"}},
			SecurityHeaders: config.SecurityHeadersConfig{
				ContentTypeNosniff: true,
				FrameOptions:       "DENY",
			},
		},
	}

	router := NewEngine(cfg, zapLoggerForTest(t), RouterOptions{})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected security headers middleware to set nosniff, got %q", got)
	}
}

func TestRouterRegistersPromptRestoreRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		App: config.AppConfig{Name: "test", Env: "test"},
		Auth: config.AuthConfig{
			AccessTokenSecret: "secret",
		},
		Server: config.ServerConfig{
			CORS: config.CORSConfig{AllowOrigins: []string{"*"}},
		},
	}

	handler := NewPromptHandler(nil)
	router := NewEngine(cfg, zapLoggerForTest(t), RouterOptions{
		PromptHandler: handler,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/prompts/123/restore", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected auth middleware to reject request with 401, got %d", w.Code)
	}
}

func zapLoggerForTest(t *testing.T) *zap.Logger {
	t.Helper()
	return zap.NewNop()
}
