package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zacharykka/prompt-manager/internal/config"
	"go.uber.org/zap"
)

// RouterOptions 用于自定义路由行为，例如注入中间件。
type RouterOptions struct {
	Middlewares   []gin.HandlerFunc
	HealthHandler gin.HandlerFunc
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
		healthHandler = func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, gin.H{
				"status":  "ok",
				"service": cfg.App.Name,
				"env":     cfg.App.Env,
			})
		}
	}

	engine.GET("/healthz", healthHandler)

	logger.Info("http router ready", zap.String("env", cfg.App.Env))

	return engine
}
