package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	authutil "github.com/zacharykka/prompt-manager/pkg/auth"
)

func TestAuthGuard_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AuthGuard("secret"))
	router.GET("/protected", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rec.Code)
	}
}

func TestAuthGuard_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AuthGuard("secret"))
	router.GET("/protected", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, ctx.GetString(UserContextKey))
	})

	token, err := authutil.GenerateToken("secret", time.Minute, authutil.Claims{
		TenantID:  "tenant",
		UserID:    "user",
		Role:      "admin",
		TokenType: "access",
	})
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rec.Code)
	}
	if rec.Body.String() != "user" {
		t.Fatalf("expected user in body got %s", rec.Body.String())
	}
}
