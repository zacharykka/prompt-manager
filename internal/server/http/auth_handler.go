package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	authsvc "github.com/zacharykka/prompt-manager/internal/service/auth"
	"github.com/zacharykka/prompt-manager/pkg/httpx"
)

// AuthHandler 处理认证相关请求。
type AuthHandler struct {
	service *authsvc.Service
}

// NewAuthHandler 构造认证处理器。
func NewAuthHandler(service *authsvc.Service) *AuthHandler {
	return &AuthHandler{service: service}
}

// RegisterRoutes 注册认证相关路由。
func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/register", h.Register)
	rg.POST("/login", h.Login)
	rg.POST("/refresh", h.Refresh)
}

type registerRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role"`
}

type loginRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Register 创建租户用户。
func (h *AuthHandler) Register(ctx *gin.Context) {
	var req registerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpx.RespondError(ctx, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), nil)
		return
	}

	user, err := h.service.Register(ctx, req.TenantID, req.Email, req.Password, req.Role)
	if err != nil {
		h.handleError(ctx, err)
		return
	}

	httpx.RespondOK(ctx, gin.H{"user": user})
}

// Login 校验凭证并返回令牌。
func (h *AuthHandler) Login(ctx *gin.Context) {
	var req loginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpx.RespondError(ctx, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), nil)
		return
	}

	tokens, user, err := h.service.Login(ctx, req.TenantID, req.Email, req.Password)
	if err != nil {
		h.handleError(ctx, err)
		return
	}

	httpx.RespondOK(ctx, gin.H{
		"tokens": tokens,
		"user":   user,
	})
}

// Refresh 使用刷新令牌颁发新访问令牌。
func (h *AuthHandler) Refresh(ctx *gin.Context) {
	var req refreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpx.RespondError(ctx, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), nil)
		return
	}

	tokens, user, err := h.service.Refresh(ctx, req.RefreshToken)
	if err != nil {
		h.handleError(ctx, err)
		return
	}

	httpx.RespondOK(ctx, gin.H{
		"tokens": tokens,
		"user":   user,
	})
}

func (h *AuthHandler) handleError(ctx *gin.Context, err error) {
	switch err {
	case authsvc.ErrTenantRequired, authsvc.ErrInvalidInput:
		httpx.RespondError(ctx, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
	case authsvc.ErrTenantNotFound:
		httpx.RespondError(ctx, http.StatusNotFound, "TENANT_NOT_FOUND", err.Error(), nil)
	case authsvc.ErrUserExists:
		httpx.RespondError(ctx, http.StatusConflict, "USER_EXISTS", err.Error(), nil)
	case authsvc.ErrInvalidCredentials:
		httpx.RespondError(ctx, http.StatusUnauthorized, "INVALID_CREDENTIALS", "邮箱或密码错误", nil)
	case authsvc.ErrUserDisabled:
		httpx.RespondError(ctx, http.StatusForbidden, "USER_DISABLED", err.Error(), nil)
	case authsvc.ErrTokenInvalid:
		httpx.RespondError(ctx, http.StatusUnauthorized, "TOKEN_INVALID", err.Error(), nil)
	default:
		httpx.RespondError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
}
