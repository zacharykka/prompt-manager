package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zacharykka/prompt-manager/internal/config"
)

// SecurityHeaders 设置通用 Web 安全响应头，减少默认配置下的安全风险。
func SecurityHeaders(cfg config.SecurityHeadersConfig) gin.HandlerFunc {
	contentSecurityPolicy := strings.TrimSpace(cfg.ContentSecurityPolicy)
	frameOptions := strings.TrimSpace(cfg.FrameOptions)
	referrerPolicy := strings.TrimSpace(cfg.ReferrerPolicy)
	xssProtection := strings.TrimSpace(cfg.XSSProtection)
	crossOriginOpenerPolicy := strings.TrimSpace(cfg.CrossOriginOpenerPolicy)
	crossOriginEmbedderPolicy := strings.TrimSpace(cfg.CrossOriginEmbedderPolicy)
	crossOriginResourcePolicy := strings.TrimSpace(cfg.CrossOriginResourcePolicy)

	return func(ctx *gin.Context) {
		headers := ctx.Writer.Header()

		if cfg.ContentTypeNosniff {
			headers.Set("X-Content-Type-Options", "nosniff")
		}
		if frameOptions != "" {
			headers.Set("X-Frame-Options", frameOptions)
		}
		if referrerPolicy != "" {
			headers.Set("Referrer-Policy", referrerPolicy)
		}
		if contentSecurityPolicy != "" {
			headers.Set("Content-Security-Policy", contentSecurityPolicy)
		}
		if xssProtection != "" {
			headers.Set("X-XSS-Protection", xssProtection)
		}
		if crossOriginOpenerPolicy != "" {
			headers.Set("Cross-Origin-Opener-Policy", crossOriginOpenerPolicy)
		}
		if crossOriginEmbedderPolicy != "" {
			headers.Set("Cross-Origin-Embedder-Policy", crossOriginEmbedderPolicy)
		}
		if crossOriginResourcePolicy != "" {
			headers.Set("Cross-Origin-Resource-Policy", crossOriginResourcePolicy)
		}

		ctx.Next()
	}
}
