package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	"github.com/zacharykka/prompt-manager/pkg/httpx"
)

// KeyFunc 提取用于限流的 key。
type KeyFunc func(*gin.Context) string

// RateLimit 返回基于 limiter 的 Gin 中间件。
func RateLimit(l *limiter.Limiter, keyFunc KeyFunc) gin.HandlerFunc {
	if keyFunc == nil {
		keyFunc = KeyByClientIP()
	}

	return func(ctx *gin.Context) {
		key := keyFunc(ctx)
		if key == "" {
			key = ctx.ClientIP()
		}

		context, err := l.Get(ctx, key)
		if err != nil {
			httpx.RespondError(ctx, http.StatusInternalServerError, "RATE_LIMIT_ERROR", err.Error(), nil)
			ctx.Abort()
			return
		}

		ctx.Writer.Header().Set("X-RateLimit-Limit", strconv.FormatInt(context.Limit, 10))
		ctx.Writer.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(context.Remaining, 10))
		ctx.Writer.Header().Set("X-RateLimit-Reset", strconv.FormatInt(context.Reset, 10))

		if context.Reached {
			httpx.RespondError(ctx, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "请求过于频繁，请稍后再试", nil)
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}

// KeyByClientIP 使用客户端 IP 作为限流 key。
func KeyByClientIP() KeyFunc {
	return func(ctx *gin.Context) string {
		return ctx.ClientIP()
	}
}

// KeyByUserOrIP 优先使用用户 ID，否则回退到 IP。
func KeyByUserOrIP() KeyFunc {
	return func(ctx *gin.Context) string {
		if userID := ctx.GetString(UserContextKey); userID != "" {
			return userID
		}
		return ctx.ClientIP()
	}
}
