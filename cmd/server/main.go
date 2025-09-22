package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
	"github.com/ulule/limiter/v3"
	memorystore "github.com/ulule/limiter/v3/drivers/store/memory"
	"github.com/zacharykka/prompt-manager/internal/app"
	"github.com/zacharykka/prompt-manager/internal/config"
	"github.com/zacharykka/prompt-manager/internal/infra"
	"github.com/zacharykka/prompt-manager/internal/middleware"
	httpserver "github.com/zacharykka/prompt-manager/internal/server/http"
	"github.com/zacharykka/prompt-manager/internal/service/auth"
	"github.com/zacharykka/prompt-manager/internal/service/prompt"
	"github.com/zacharykka/prompt-manager/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	opts := parseFlags()

	cfg, err := config.Load(opts.ConfigDir, opts.Env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.Logging.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = log.Sync()
	}()

	initCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	infraContainer, cleanup, err := infra.Initialize(initCtx, cfg, log)
	cancel()
	if err != nil {
		log.Fatal("依赖初始化失败", zap.Error(err))
	}

	defer func() {
		if cleanup == nil {
			return
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := cleanup(shutdownCtx); err != nil {
			log.Warn("资源清理失败", zap.Error(err))
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	authService := auth.NewService(infraContainer.Repos, cfg.Auth)
	authHandler := httpserver.NewAuthHandler(authService)
	promptService := prompt.NewService(infraContainer.Repos)
	promptHandler := httpserver.NewPromptHandler(promptService)

	store := memorystore.NewStore()
	generalLimiter := middleware.RateLimit(limiter.New(store, limiter.Rate{Period: time.Minute, Limit: 120}), middleware.KeyByClientIP())
	loginLimiter := middleware.RateLimit(limiter.New(store, limiter.Rate{Period: time.Minute, Limit: 10}), middleware.KeyByClientIP())

	engine := httpserver.NewEngine(cfg, log, httpserver.RouterOptions{
		Middlewares: []gin.HandlerFunc{
			middleware.RequestLogger(log),
		},
		HealthDeps: &httpserver.HealthDependencies{
			DB:    infraContainer.DB,
			Redis: infraContainer.Redis,
		},
		AuthHandler:    authHandler,
		PromptHandler:  promptHandler,
		RateLimiter:    generalLimiter,
		LoginRateLimit: loginLimiter,
	})

	application := app.New(cfg, log, engine)

	if err := application.Run(ctx); err != nil {
		log.Fatal("服务运行异常", zap.Error(err))
	}
}

// options 控制命令行参数。
type options struct {
	ConfigDir string
	Env       string
}

func parseFlags() options {
	var opts options
	pflag.StringVar(&opts.ConfigDir, "config-dir", "./config", "配置文件目录")
	pflag.StringVar(&opts.Env, "env", "", "强制指定运行环境，覆盖 PROMPT_MANAGER_ENV")
	pflag.Parse()
	return opts
}
