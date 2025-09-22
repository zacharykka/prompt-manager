package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLimitRequestBody_AllowsWithinLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(LimitRequestBody(16))
	router.POST("/", func(ctx *gin.Context) {
		var payload map[string]any
		if err := ctx.ShouldBindJSON(&payload); err != nil {
			ctx.AbortWithStatus(http.StatusBadRequest)
			return
		}
		ctx.Status(http.StatusOK)
	})

	body := bytes.NewBufferString("{\"a\":1}")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rec.Code)
	}
}

func TestLimitRequestBody_RejectsTooLarge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(LimitRequestBody(10))
	router.POST("/", func(ctx *gin.Context) {
		var payload map[string]any
		if err := ctx.ShouldBindJSON(&payload); err != nil {
			ctx.AbortWithStatus(http.StatusBadRequest)
			return
		}
		ctx.Status(http.StatusOK)
	})

	body := bytes.NewBufferString("{\"payload\":\"0123456789\"}")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", rec.Code)
	}
}
