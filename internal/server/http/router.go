package http

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/zacharykka/prompt-manager/internal/config"
	"github.com/zacharykka/prompt-manager/internal/infra/cache"
	"github.com/zacharykka/prompt-manager/internal/infra/database"
	"github.com/zacharykka/prompt-manager/internal/middleware"
	"go.uber.org/zap"
)

// HealthDependencies 汇总健康检查所需的依赖。
type HealthDependencies struct {
	DB    *sql.DB
	Redis *redis.Client
}

// RouterOptions 用于自定义路由行为，例如注入中间件。
type RouterOptions struct {
	Middlewares   []gin.HandlerFunc
	HealthHandler gin.HandlerFunc
	HealthDeps    *HealthDependencies
	AuthHandler   *AuthHandler
	PromptHandler *PromptHandler
}

// NewEngine 根据环境配置初始化 Gin 引擎，并注册基础路由。
func NewEngine(cfg *config.Config, logger *zap.Logger, opts RouterOptions) *gin.Engine {
	ginMode := gin.DebugMode
	if cfg.App.Env == "production" {
		ginMode = gin.ReleaseMode
	}
	gin.SetMode(ginMode)

	engine := gin.New()

	engine.Use(gin.Recovery())

	for _, mw := range opts.Middlewares {
		if mw != nil {
			engine.Use(mw)
		}
	}

	healthHandler := opts.HealthHandler
	if healthHandler == nil {
		healthHandler = defaultHealthHandler(cfg, opts.HealthDeps)
	}

	engine.GET("/healthz", healthHandler)

	api := engine.Group("/api/v1")
	if opts.AuthHandler != nil {
		authGroup := api.Group("/auth")
		opts.AuthHandler.RegisterRoutes(authGroup)
	}
	if opts.PromptHandler != nil {
		promptGroup := api.Group("/prompts")
		promptGroup.Use(middleware.AuthGuard(cfg.Auth.AccessTokenSecret))

		// Read-only endpoints available to all authenticated roles
		promptGroup.GET("/", opts.PromptHandler.ListPrompts)
		promptGroup.GET("/:id", opts.PromptHandler.GetPrompt)
		promptGroup.GET("/:id/versions", opts.PromptHandler.ListPromptVersions)

		// Write endpoints restricted to admin/editor
		writeGroup := promptGroup.Group("")
		writeGroup.Use(middleware.RequireRoles(middleware.RoleAdmin, middleware.RoleEditor))
		writeGroup.POST("/", opts.PromptHandler.CreatePrompt)
		writeGroup.POST("/:id/versions", opts.PromptHandler.CreatePromptVersion)
		writeGroup.POST("/:id/versions/:versionId/activate", opts.PromptHandler.SetActiveVersion)
	}

	logger.Info("http router ready", zap.String("env", cfg.App.Env))

	return engine
}

func defaultHealthHandler(cfg *config.Config, deps *HealthDependencies) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		httpStatus := http.StatusOK
		result := gin.H{
			"status":  "ok",
			"service": cfg.App.Name,
			"env":     cfg.App.Env,
		}

		if deps != nil {
			dependencies := gin.H{}
			if deps.DB != nil {
				if err := database.Health(ctx.Request.Context(), deps.DB); err != nil {
					httpStatus = http.StatusServiceUnavailable
					result["status"] = "degraded"
					dependencies["database"] = gin.H{
						"status": "error",
						"error":  err.Error(),
					}
				} else {
					dependencies["database"] = gin.H{"status": "ok"}
				}
			} else {
				dependencies["database"] = gin.H{"status": "missing"}
			}

			if deps.Redis != nil {
				if err := cache.Health(ctx.Request.Context(), deps.Redis); err != nil {
					httpStatus = http.StatusServiceUnavailable
					result["status"] = "degraded"
					dependencies["redis"] = gin.H{
						"status": "error",
						"error":  err.Error(),
					}
				} else {
					dependencies["redis"] = gin.H{"status": "ok"}
				}
			} else {
				dependencies["redis"] = gin.H{"status": "missing"}
			}

			result["dependencies"] = dependencies
		}

		ctx.JSON(httpStatus, result)
	}
}
