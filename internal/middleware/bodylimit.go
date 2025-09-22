package middleware

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

// LimitRequestBody 限制请求体大小，超出时返回 413。
func LimitRequestBody(maxBytes int64) gin.HandlerFunc {
    return func(ctx *gin.Context) {
        if maxBytes > 0 {
            ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxBytes)
        }
        ctx.Next()
    }
}
