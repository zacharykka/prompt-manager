package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zacharykka/prompt-manager/internal/config"
	"go.uber.org/zap"
)

// Application 负责组织配置、日志与 HTTP Server 的生命周期。
type Application struct {
	cfg    *config.Config
	logger *zap.Logger
	engine *gin.Engine
	server *http.Server
}

// New 构建应用实例，并初始化 HTTP 服务配置。
func New(cfg *config.Config, logger *zap.Logger, engine *gin.Engine) *Application {
	httpServer := &http.Server{
		Addr:              cfg.Server.Addr(),
		Handler:           engine,
		ReadHeaderTimeout: cfg.Server.ReadTimeout,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
	}

	return &Application{
		cfg:    cfg,
		logger: logger,
		engine: engine,
		server: httpServer,
	}
}

// Run 启动 HTTP 服务并监听上下文取消，实现优雅退出。
func (a *Application) Run(ctx context.Context) error {
	a.logger.Info("starting http server", zap.String("addr", a.server.Addr))

	errCh := make(chan error, 1)
	go func() {
		if err := a.server.ListenAndServe(); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.Server.ShutdownTimeout)
		defer cancel()
		return a.shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// shutdown 执行优雅停机逻辑。
func (a *Application) shutdown(ctx context.Context) error {
	a.logger.Info("shutting down http server")
	if err := a.server.Shutdown(ctx); err != nil {
		a.logger.Error("graceful shutdown failed", zap.Error(err))
		return err
	}
	a.logger.Info("shutdown complete")
	return nil
}

// Engine 暴露 Gin 引擎实例，方便注册额外路由。
func (a *Application) Engine() *gin.Engine {
	return a.engine
}

// WaitForShutdownDelay 用于在需要时等待 ShutdownTimeout，以保证 goroutine 收敛。
func (a *Application) WaitForShutdownDelay() {
	time.Sleep(a.cfg.Server.ShutdownTimeout)
}
