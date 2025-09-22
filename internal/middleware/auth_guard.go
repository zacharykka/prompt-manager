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

// Roles 定义可用角色名称。
const (
	RoleAdmin  = "admin"
	RoleEditor = "editor"
	RoleViewer = "viewer"
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

		ctx.Set(UserContextKey, claims.UserID)
		ctx.Set(UserRoleContextKey, claims.Role)
		ctx.Set("auth_claims", claims)
		ctx.Next()
	}
}

// RequireRoles 验证当前用户是否具备指定角色之一。
func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[strings.ToLower(role)] = struct{}{}
	}

	return func(ctx *gin.Context) {
		role := strings.ToLower(ctx.GetString(UserRoleContextKey))
		if _, ok := allowed[role]; !ok {
			httpx.RespondError(ctx, http.StatusForbidden, "FORBIDDEN", "当前角色无权限执行该操作", nil)
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}
