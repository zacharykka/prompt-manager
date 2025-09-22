package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	memorystore "github.com/ulule/limiter/v3/drivers/store/memory"
)

func TestRateLimit_AllowsWithinLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := memorystore.NewStore()
	l := limiter.New(store, limiter.Rate{Period: time.Minute, Limit: 2})

	router := gin.New()
	router.Use(RateLimit(l, KeyByClientIP()))
	router.GET("/", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rec.Code)
	}

	if rec.Header().Get("X-RateLimit-Limit") == "" {
		t.Fatalf("expected rate limit headers")
	}
}

func TestRateLimit_BlocksWhenExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := memorystore.NewStore()
	l := limiter.New(store, limiter.Rate{Period: time.Hour, Limit: 1})

	router := gin.New()
	router.Use(RateLimit(l, KeyByClientIP()))
	router.GET("/", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// first request allowed
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first request should pass, got %d", rec1.Code)
	}

	// second request blocked
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 got %d", rec2.Code)
	}
}

func TestRateLimit_KeyByUserOrIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := memorystore.NewStore()
	l := limiter.New(store, limiter.Rate{Period: time.Minute, Limit: 1})

	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set(UserContextKey, "user-123")
		ctx.Next()
	})
	router.Use(RateLimit(l, KeyByUserOrIP()))
	router.GET("/", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first request should pass, got %d", rec1.Code)
	}

	// use different IP but same user -> still blocked
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-Forwarded-For", "203.0.113.1")
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request should be limited by user key, got %d", rec2.Code)
	}
}

func TestRateLimit_CustomKeyFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := memorystore.NewStore()
	l := limiter.New(store, limiter.Rate{Period: time.Second, Limit: 1})

	router := gin.New()
	router.Use(RateLimit(l, func(*gin.Context) string { return "" }))
	router.GET("/", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected success, got %d", rec.Code)
	}
}
