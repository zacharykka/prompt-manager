package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zacharykka/prompt-manager/internal/middleware"
	promptsvc "github.com/zacharykka/prompt-manager/internal/service/prompt"
	"github.com/zacharykka/prompt-manager/pkg/httpx"
)

// PromptHandler 处理 Prompt 相关 HTTP 请求。
type PromptHandler struct {
	service *promptsvc.Service
}

// NewPromptHandler 创建 PromptHandler。
func NewPromptHandler(service *promptsvc.Service) *PromptHandler {
	return &PromptHandler{service: service}
}

// RegisterRoutes 注册 Prompt 相关路由。
func (h *PromptHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/", h.CreatePrompt)
	rg.GET("/", h.ListPrompts)
	rg.GET("/:id", h.GetPrompt)
	rg.POST("/:id/versions", h.CreatePromptVersion)
	rg.GET("/:id/versions", h.ListPromptVersions)
	rg.POST("/:id/versions/:versionId/activate", h.SetActiveVersion)
}

type createPromptRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description *string  `json:"description"`
	Tags        []string `json:"tags"`
}

type createPromptVersionRequest struct {
	Body            string      `json:"body" binding:"required"`
	VariablesSchema interface{} `json:"variables_schema"`
	Metadata        interface{} `json:"metadata"`
	Status          string      `json:"status"`
	Activate        bool        `json:"activate"`
}

// CreatePrompt 处理创建 Prompt 请求。
func (h *PromptHandler) CreatePrompt(ctx *gin.Context) {
	var req createPromptRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpx.RespondError(ctx, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), nil)
		return
	}

	createdBy := ctx.GetString(middleware.UserContextKey)

	prompt, err := h.service.CreatePrompt(ctx, promptsvc.CreatePromptInput{
		Name:        req.Name,
		Description: req.Description,
		Tags:        req.Tags,
		CreatedBy:   createdBy,
	})
	if err != nil {
		h.handleError(ctx, err)
		return
	}

	httpx.RespondOK(ctx, gin.H{"prompt": prompt})
}

// ListPrompts 列出 Prompt。
func (h *PromptHandler) ListPrompts(ctx *gin.Context) {
	limit, offset := parsePagination(ctx.Query("limit"), ctx.Query("offset"))

	prompts, err := h.service.ListPrompts(ctx, limit, offset)
	if err != nil {
		httpx.RespondError(ctx, http.StatusInternalServerError, "LIST_FAILED", err.Error(), nil)
		return
	}

	httpx.RespondOK(ctx, gin.H{"items": prompts})
}

// GetPrompt 获取指定 Prompt。
func (h *PromptHandler) GetPrompt(ctx *gin.Context) {
	prompt, err := h.service.GetPrompt(ctx, ctx.Param("id"))
	if err != nil {
		h.handleError(ctx, err)
		return
	}

	httpx.RespondOK(ctx, gin.H{"prompt": prompt})
}

// CreatePromptVersion 创建新的 Prompt 版本。
func (h *PromptHandler) CreatePromptVersion(ctx *gin.Context) {
	var req createPromptVersionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpx.RespondError(ctx, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), nil)
		return
	}

	createdBy := ctx.GetString(middleware.UserContextKey)

	version, err := h.service.CreatePromptVersion(ctx, promptsvc.CreatePromptVersionInput{
		PromptID:        ctx.Param("id"),
		Body:            req.Body,
		VariablesSchema: req.VariablesSchema,
		Metadata:        req.Metadata,
		Status:          req.Status,
		CreatedBy:       createdBy,
		Activate:        req.Activate,
	})
	if err != nil {
		h.handleError(ctx, err)
		return
	}

	httpx.RespondOK(ctx, gin.H{"version": version})
}

// ListPromptVersions 列出 Prompt 的版本。
func (h *PromptHandler) ListPromptVersions(ctx *gin.Context) {
	limit, offset := parsePagination(ctx.Query("limit"), ctx.Query("offset"))

	versions, err := h.service.ListPromptVersions(ctx, ctx.Param("id"), limit, offset)
	if err != nil {
		h.handleError(ctx, err)
		return
	}

	httpx.RespondOK(ctx, gin.H{"items": versions})
}

// SetActiveVersion 设定当前使用的版本。
func (h *PromptHandler) SetActiveVersion(ctx *gin.Context) {
	promptID := ctx.Param("id")
	versionID := ctx.Param("versionId")

	if err := h.service.SetActiveVersion(ctx, promptID, versionID); err != nil {
		h.handleError(ctx, err)
		return
	}

	httpx.RespondOK(ctx, gin.H{"prompt_id": promptID, "active_version_id": versionID})
}

func (h *PromptHandler) handleError(ctx *gin.Context, err error) {
	switch err {
	case promptsvc.ErrNameRequired, promptsvc.ErrBodyRequired:
		httpx.RespondError(ctx, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
	case promptsvc.ErrPromptAlreadyExists:
		httpx.RespondError(ctx, http.StatusConflict, "PROMPT_EXISTS", err.Error(), nil)
	case promptsvc.ErrPromptNotFound:
		httpx.RespondError(ctx, http.StatusNotFound, "PROMPT_NOT_FOUND", err.Error(), nil)
	case promptsvc.ErrVersionNotFound:
		httpx.RespondError(ctx, http.StatusNotFound, "VERSION_NOT_FOUND", err.Error(), nil)
	default:
		httpx.RespondError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
}

func parsePagination(limitStr, offsetStr string) (int, int) {
	limit := 50
	offset := 0

	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}
	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}
	return limit, offset
}
