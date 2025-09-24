package http

import (
	"database/sql"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
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
	Middlewares    []gin.HandlerFunc
	HealthHandler  gin.HandlerFunc
	HealthDeps     *HealthDependencies
	AuthHandler    *AuthHandler
	PromptHandler  *PromptHandler
	RateLimiter    gin.HandlerFunc
	AuthRateLimit  gin.HandlerFunc
	LoginRateLimit gin.HandlerFunc
}

// NewEngine 根据环境配置初始化 Gin 引擎，并注册基础路由。
func NewEngine(cfg *config.Config, logger *zap.Logger, opts RouterOptions) *gin.Engine {
	ginMode := gin.DebugMode
	if cfg.App.Env == "production" {
		ginMode = gin.ReleaseMode
	}
	gin.SetMode(ginMode)

	engine := gin.New()
	engine.RedirectTrailingSlash = false

	engine.Use(gin.Recovery())
	engine.Use(middleware.SecurityHeaders(cfg.Server.SecurityHeaders))
	if cfg.Server.MaxRequestBody > 0 {
		engine.MaxMultipartMemory = cfg.Server.MaxRequestBody
		engine.Use(middleware.LimitRequestBody(cfg.Server.MaxRequestBody))
	}
	engine.Use(cors.New(buildCORSConfig(cfg.Server)))

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
	if opts.RateLimiter != nil {
		api.Use(opts.RateLimiter)
	}
	if opts.AuthHandler != nil {
		authGroup := api.Group("/auth")
		if opts.AuthRateLimit != nil {
			authGroup.Use(opts.AuthRateLimit)
		}
		authGroup.POST("/register", opts.AuthHandler.Register)
		if opts.LoginRateLimit != nil {
			authGroup.POST("/login", opts.LoginRateLimit, opts.AuthHandler.Login)
		} else {
			authGroup.POST("/login", opts.AuthHandler.Login)
		}
		authGroup.POST("/refresh", opts.AuthHandler.Refresh)
	}
	if opts.PromptHandler != nil {
		promptGroup := api.Group("/prompts")
		promptGroup.Use(middleware.AuthGuard(cfg.Auth.AccessTokenSecret))
		promptGroup.GET("", opts.PromptHandler.ListPrompts)
		promptGroup.GET("/", opts.PromptHandler.ListPrompts)
		promptGroup.GET("/:id", opts.PromptHandler.GetPrompt)
		promptGroup.GET("/:id/versions", opts.PromptHandler.ListPromptVersions)
		promptGroup.GET("/:id/versions/:versionId/diff", opts.PromptHandler.DiffPromptVersion)
		promptGroup.GET("/:id/stats", opts.PromptHandler.GetPromptStats)

		writeGroup := promptGroup.Group("")
		writeGroup.Use(middleware.RequireRoles(middleware.RoleAdmin, middleware.RoleEditor))
		writeGroup.POST("", opts.PromptHandler.CreatePrompt)
		writeGroup.POST("/", opts.PromptHandler.CreatePrompt)
		writeGroup.PUT("/:id", opts.PromptHandler.UpdatePrompt)
		writeGroup.PATCH("/:id", opts.PromptHandler.UpdatePrompt)
		writeGroup.POST("/:id/versions", opts.PromptHandler.CreatePromptVersion)
		writeGroup.POST("/:id/versions/:versionId/activate", opts.PromptHandler.SetActiveVersion)
		writeGroup.DELETE("/:id", opts.PromptHandler.DeletePrompt)
		writeGroup.POST("/:id/restore", opts.PromptHandler.RestorePrompt)
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

func buildCORSConfig(serverCfg config.ServerConfig) cors.Config {
	config := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"},
		AllowCredentials: serverCfg.CORS.AllowCredentials,
		MaxAge:           12 * time.Hour,
	}

	exactOrigins, patternOrigins, allowAll := classifyAllowedOrigins(serverCfg.CORS.AllowOrigins)
	switch {
	case allowAll:
		config.AllowAllOrigins = true
	case len(patternOrigins) == 0:
		config.AllowOrigins = exactOrigins
	default:
		config.AllowOriginFunc = func(origin string) bool {
			if origin == "" {
				return false
			}
			for _, allowed := range exactOrigins {
				if strings.EqualFold(origin, allowed) {
					return true
				}
			}
			for _, re := range patternOrigins {
				if re.MatchString(origin) {
					return true
				}
			}
			return false
		}
	}

	return config
}

func classifyAllowedOrigins(origins []string) (exact []string, patterns []*regexp.Regexp, allowAll bool) {
	throttled := make(map[string]struct{})
	for _, origin := range origins {
		clean := strings.TrimSpace(origin)
		if clean == "" {
			continue
		}
		if clean == "*" {
			return nil, nil, true
		}
		if strings.Contains(clean, "*") {
			pattern := regexp.QuoteMeta(clean)
			pattern = strings.ReplaceAll(pattern, "\\*", ".*")
			re, err := regexp.Compile("^" + pattern + "$")
			if err != nil {
				continue
			}
			patterns = append(patterns, re)
			continue
		}
		key := strings.ToLower(clean)
		if _, ok := throttled[key]; ok {
			continue
		}
		throttled[key] = struct{}{}
		exact = append(exact, clean)
	}
	return exact, patterns, false
}
