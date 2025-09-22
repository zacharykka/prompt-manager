package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	authutil "github.com/zacharykka/prompt-manager/pkg/auth"
	"github.com/zacharykka/prompt-manager/pkg/httpx"
)

const (
	// UserContextKey 在上下文中存储用户 ID。
	UserContextKey = "user_id"
	// UserRoleContextKey 在上下文中存储用户角色。
	UserRoleContextKey = "user_role"
)

// AuthGuard 校验 Bearer Token 并注入用户/租户信息。
func AuthGuard(accessSecret string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		header := ctx.GetHeader("Authorization")
		if header == "" {
			httpx.RespondError(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "缺少认证信息", nil)
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			httpx.RespondError(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "认证信息格式错误", nil)
			return
		}

		claims, err := authutil.ParseToken(parts[1], accessSecret)
		if err != nil || claims.TokenType != "access" {
			httpx.RespondError(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "令牌无效", nil)
			return
		}

		ctx.Set(TenantContextKey, claims.TenantID)
		ctx.Set(UserContextKey, claims.UserID)
		ctx.Set(UserRoleContextKey, claims.Role)
		ctx.Set("auth_claims", claims)
		ctx.Next()
	}
}
