package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zacharykka/prompt-manager/pkg/httpx"
)

const (
	// TenantContextKey 是在 Gin Context 中存储租户信息的键名。
	TenantContextKey = "tenant_id"
	tenantHeader     = "X-Tenant-ID"
	defaultTenant    = "default"
)

// TenantInjector 提供基础的租户注入逻辑，后续可替换为 JWT/OIDC 解析。
func TenantInjector() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if tenantID, exists := ctx.Get(TenantContextKey); exists {
			if idStr, ok := tenantID.(string); ok && idStr != "" {
				ctx.Writer.Header().Set(tenantHeader, idStr)
				ctx.Next()
				return
			}
		}
		tenantID := ctx.GetHeader(tenantHeader)
		if tenantID == "" {
			tenantID = defaultTenant
		}
		ctx.Set(TenantContextKey, tenantID)
		ctx.Writer.Header().Set(tenantHeader, tenantID)
		ctx.Next()
	}
}

// RequireTenant 可用于确保请求携带租户信息。
func RequireTenant() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if tenant := GetTenantID(ctx); tenant == "" {
			httpx.RespondError(ctx, http.StatusUnauthorized, "TENANT_MISSING", "缺少租户标识", nil)
			return
		}
		ctx.Next()
	}
}

// GetTenantID 从上下文读取租户标识。
func GetTenantID(ctx *gin.Context) string {
	val, ok := ctx.Get(TenantContextKey)
	if !ok {
		return ""
	}
	if tenant, ok := val.(string); ok {
		return tenant
	}
	return ""
}
