package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTenantInjector_DefaultTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TenantInjector())
	router.GET("/test", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, GetTenantID(ctx))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d got %d", http.StatusOK, rec.Code)
	}
	if body := rec.Body.String(); body != "default" {
		t.Fatalf("expected tenant \"default\" got %q", body)
	}
	if header := rec.Header().Get("X-Tenant-ID"); header != "default" {
		t.Fatalf("expected header X-Tenant-ID to be default got %q", header)
	}
}

func TestRequireTenant_Missing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/secure", RequireTenant(), func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d got %d", http.StatusUnauthorized, rec.Code)
	}
	expected := `{"code":"TENANT_MISSING","message":"缺少租户标识"}`
	if rec.Body.String() != expected {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
