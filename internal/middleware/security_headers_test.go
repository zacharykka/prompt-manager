package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zacharykka/prompt-manager/internal/config"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.SecurityHeadersConfig{
		FrameOptions:              "DENY",
		ContentTypeNosniff:        true,
		ReferrerPolicy:            "no-referrer",
		XSSProtection:             "0",
		ContentSecurityPolicy:     "default-src 'self'",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginResourcePolicy: "same-site",
	}

	router := gin.New()
	router.Use(SecurityHeaders(cfg))
	router.GET("/", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "ok")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	router.ServeHTTP(recorder, request)

	headers := recorder.Header()
	tests := map[string]string{
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
		"Referrer-Policy":              "no-referrer",
		"X-XSS-Protection":             "0",
		"Content-Security-Policy":      "default-src 'self'",
		"Cross-Origin-Opener-Policy":   "same-origin",
		"Cross-Origin-Embedder-Policy": "require-corp",
		"Cross-Origin-Resource-Policy": "same-site",
	}

	for key, expected := range tests {
		if got := headers.Get(key); got != expected {
			t.Fatalf("expected %s header %q got %q", key, expected, got)
		}
	}

	if value := headers.Get("Strict-Transport-Security"); value != "" {
		t.Fatalf("expected Strict-Transport-Security header to be empty, got %q", value)
	}
}
