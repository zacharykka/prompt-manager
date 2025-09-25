package http

import (
	"net/http"
	"strconv"
	"strings"

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
	rg.POST("", h.CreatePrompt)
	rg.POST("/", h.CreatePrompt)
	rg.GET("", h.ListPrompts)
	rg.GET("/", h.ListPrompts)
	rg.GET("/:id", h.GetPrompt)
	rg.PUT("/:id", h.UpdatePrompt)
	rg.PATCH("/:id", h.UpdatePrompt)
	rg.POST("/:id/versions", h.CreatePromptVersion)
	rg.GET("/:id/versions", h.ListPromptVersions)
	rg.GET("/:id/versions/:versionId/diff", h.DiffPromptVersion)
	rg.POST("/:id/versions/:versionId/activate", h.SetActiveVersion)
	rg.GET("/:id/stats", h.GetPromptStats)
	rg.DELETE("/:id", h.DeletePrompt)
	rg.POST("/:id/restore", h.RestorePrompt)
}

type createPromptRequest struct {
	Name        string   `json:"name" binding:"required,min=1,max=128"`
	Description *string  `json:"description"`
	Tags        []string `json:"tags" binding:"max=10"`
	Body        string   `json:"body" binding:"omitempty,min=1"`
}

type updatePromptRequest struct {
	Name        *string   `json:"name" binding:"omitempty,min=1,max=128"`
	Description *string   `json:"description"`
	Tags        *[]string `json:"tags" binding:"max=10"`
}

type createPromptVersionRequest struct {
	Body            string      `json:"body" binding:"required,min=1"`
	VariablesSchema interface{} `json:"variables_schema"`
	Metadata        interface{} `json:"metadata"`
	Status          string      `json:"status" binding:"omitempty,oneof=draft published archived"`
	Activate        bool        `json:"activate"`
}

// CreatePrompt 处理创建 Prompt 请求。
func (h *PromptHandler) CreatePrompt(ctx *gin.Context) {
	var req createPromptRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpx.RespondError(ctx, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), nil)
		return
	}

	createdBy := ctx.GetString(middleware.UserEmailContextKey)
	if createdBy == "" {
		createdBy = ctx.GetString(middleware.UserContextKey)
	}

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

	body := strings.TrimSpace(req.Body)
	if body != "" {
		if _, err := h.service.CreatePromptVersion(ctx, promptsvc.CreatePromptVersionInput{
			PromptID:  prompt.ID,
			Body:      body,
			Status:    "published",
			CreatedBy: createdBy,
			Activate:  true,
		}); err != nil {
			httpx.RespondError(ctx, http.StatusInternalServerError, "CREATE_VERSION_FAILED", err.Error(), nil)
			return
		}
		// 重新加载 Prompt 以便带上最新的激活版本信息
		updatedPrompt, err := h.service.GetPrompt(ctx, prompt.ID)
		if err == nil {
			prompt = updatedPrompt
		}
	}

	httpx.RespondOK(ctx, gin.H{"prompt": prompt})
}

// UpdatePrompt 处理更新 Prompt 请求。
func (h *PromptHandler) UpdatePrompt(ctx *gin.Context) {
	var req updatePromptRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpx.RespondError(ctx, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), nil)
		return
	}

	if req.Name == nil && req.Description == nil && req.Tags == nil {
		httpx.RespondError(ctx, http.StatusBadRequest, "INVALID_PAYLOAD", "至少需要提供一个需要更新的字段", nil)
		return
	}

	updated, err := h.service.UpdatePrompt(ctx, promptsvc.UpdatePromptInput{
		PromptID:    ctx.Param("id"),
		Name:        req.Name,
		Description: req.Description,
		Tags:        req.Tags,
	})
	if err != nil {
		h.handleError(ctx, err)
		return
	}

	httpx.RespondOK(ctx, gin.H{"prompt": updated})
}

// ListPrompts 列出 Prompt。
func (h *PromptHandler) ListPrompts(ctx *gin.Context) {
	limit, offset := parsePagination(ctx.Query("limit"), ctx.Query("offset"))
	search := strings.TrimSpace(ctx.Query("search"))

	includeDeleted := false
	if value := strings.TrimSpace(ctx.Query("includeDeleted")); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			includeDeleted = parsed
		}
	}

	prompts, total, err := h.service.ListPrompts(ctx, promptsvc.ListPromptsOptions{
		Limit:          limit,
		Offset:         offset,
		Search:         search,
		IncludeDeleted: includeDeleted,
	})
	if err != nil {
		httpx.RespondError(ctx, http.StatusInternalServerError, "LIST_FAILED", err.Error(), nil)
		return
	}

	httpx.RespondOK(ctx, gin.H{
		"items": prompts,
		"meta": gin.H{
			"total":   total,
			"limit":   limit,
			"offset":  offset,
			"hasMore": int64(offset)+int64(len(prompts)) < total,
		},
	})
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

	createdBy := ctx.GetString(middleware.UserEmailContextKey)
	if createdBy == "" {
		createdBy = ctx.GetString(middleware.UserContextKey)
	}

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
    status := strings.TrimSpace(ctx.Query("status"))

    page, err := h.service.ListPromptVersionsEx(ctx, ctx.Param("id"), limit, offset, status)
    if err != nil {
        h.handleError(ctx, err)
        return
    }

    httpx.RespondOK(ctx, gin.H{
        "items": page.Items,
        "meta": gin.H{
            "limit":   page.Limit,
            "offset":  page.Offset,
            "has_more": page.HasMore,
            "total":    page.Total,
            "pages":    page.Pages,
        },
    })
}

// DiffPromptVersion 对比指定 Prompt 版本与目标版本差异。
func (h *PromptHandler) DiffPromptVersion(ctx *gin.Context) {
	compareTo := strings.TrimSpace(strings.ToLower(ctx.Query("compareTo")))
	targetID := strings.TrimSpace(ctx.Query("targetVersionId"))

	options := promptsvc.DiffPromptVersionOptions{}
	if targetID != "" {
		options.TargetVersionID = &targetID
	} else if compareTo == "active" {
		options.CompareToActive = true
	} else {
		options.CompareToPrevious = true
	}

	diff, err := h.service.DiffPromptVersion(ctx, ctx.Param("id"), ctx.Param("versionId"), options)
	if err != nil {
		h.handleError(ctx, err)
		return
	}

	httpx.RespondOK(ctx, gin.H{"diff": diff})
}

// SetActiveVersion 设定当前使用的版本。
func (h *PromptHandler) SetActiveVersion(ctx *gin.Context) {
	promptID := ctx.Param("id")
	versionID := ctx.Param("versionId")
	activatedBy := ctx.GetString(middleware.UserEmailContextKey)
	if activatedBy == "" {
		activatedBy = ctx.GetString(middleware.UserContextKey)
	}

	if err := h.service.SetActiveVersion(ctx, promptID, versionID, activatedBy); err != nil {
		h.handleError(ctx, err)
		return
	}

	httpx.RespondOK(ctx, gin.H{"prompt_id": promptID, "active_version_id": versionID})
}

// GetPromptStats 返回执行统计数据。
func (h *PromptHandler) GetPromptStats(ctx *gin.Context) {
	days := parseQueryInt(ctx.Query("days"), 7)

	stats, err := h.service.GetExecutionStats(ctx, ctx.Param("id"), days)
	if err != nil {
		h.handleError(ctx, err)
		return
	}

	httpx.RespondOK(ctx, gin.H{"items": stats})
}

// DeletePrompt 删除指定 Prompt。
func (h *PromptHandler) DeletePrompt(ctx *gin.Context) {
	deletedBy := ctx.GetString(middleware.UserEmailContextKey)
	if deletedBy == "" {
		deletedBy = ctx.GetString(middleware.UserContextKey)
	}
	if err := h.service.DeletePrompt(ctx, ctx.Param("id"), deletedBy); err != nil {
		h.handleError(ctx, err)
		return
	}

	httpx.RespondOK(ctx, gin.H{"prompt_id": ctx.Param("id")})
}

// RestorePrompt 恢复软删除的 Prompt。
func (h *PromptHandler) RestorePrompt(ctx *gin.Context) {
	restoredBy := ctx.GetString(middleware.UserEmailContextKey)
	if restoredBy == "" {
		restoredBy = ctx.GetString(middleware.UserContextKey)
	}

	restored, err := h.service.RestorePrompt(ctx, ctx.Param("id"), restoredBy)
	if err != nil {
		h.handleError(ctx, err)
		return
	}

	httpx.RespondOK(ctx, gin.H{"prompt": restored})
}

func (h *PromptHandler) handleError(ctx *gin.Context, err error) {
	switch err {
	case promptsvc.ErrNameRequired, promptsvc.ErrBodyRequired:
		httpx.RespondError(ctx, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
	case promptsvc.ErrPromptAlreadyExists:
		httpx.RespondError(ctx, http.StatusConflict, "PROMPT_EXISTS", err.Error(), nil)
	case promptsvc.ErrPromptNotDeleted:
		httpx.RespondError(ctx, http.StatusBadRequest, "PROMPT_NOT_DELETED", err.Error(), nil)
	case promptsvc.ErrPromptNotFound:
		httpx.RespondError(ctx, http.StatusNotFound, "PROMPT_NOT_FOUND", err.Error(), nil)
	case promptsvc.ErrVersionNotFound:
		httpx.RespondError(ctx, http.StatusNotFound, "VERSION_NOT_FOUND", err.Error(), nil)
	case promptsvc.ErrNoFieldsToUpdate:
		httpx.RespondError(ctx, http.StatusBadRequest, "NO_FIELDS_TO_UPDATE", err.Error(), nil)
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

func parseQueryInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	if v, err := strconv.Atoi(value); err == nil && v > 0 {
		return v
	}
	return fallback
}
