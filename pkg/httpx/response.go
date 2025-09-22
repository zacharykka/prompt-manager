package httpx

import "github.com/gin-gonic/gin"

// SuccessResponse 标准成功响应结构。
type SuccessResponse struct {
	Data interface{} `json:"data,omitempty"`
}

// ErrorResponse 标准错误响应结构。
type ErrorResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// RespondOK 输出成功响应。
func RespondOK(ctx *gin.Context, data interface{}) {
	ctx.JSON(200, SuccessResponse{Data: data})
}

// RespondError 输出错误响应并终止处理流程。
func RespondError(ctx *gin.Context, status int, code string, message string, details interface{}) {
	ctx.AbortWithStatusJSON(status, ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	})
}
