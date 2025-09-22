package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestLogger 负责记录每一次 HTTP 请求，便于排查与审计。
func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()

		duration := time.Since(start)

		logger.Info("http request",
			zap.String("method", ctx.Request.Method),
			zap.String("path", ctx.FullPath()),
			zap.Int("status", ctx.Writer.Status()),
			zap.Duration("duration", duration),
			zap.String("client_ip", ctx.ClientIP()),
			zap.Int("size", ctx.Writer.Size()),
		)
	}
}
