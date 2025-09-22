package infra

import (
	"context"
	"database/sql"

	"github.com/redis/go-redis/v9"
	"github.com/zacharykka/prompt-manager/internal/config"
	"github.com/zacharykka/prompt-manager/internal/domain"
	"github.com/zacharykka/prompt-manager/internal/infra/cache"
	"github.com/zacharykka/prompt-manager/internal/infra/database"
	"github.com/zacharykka/prompt-manager/internal/infra/repository"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

// Container 持有应用依赖资源，负责集中关闭。
type Container struct {
	DB    *sql.DB
	Redis *redis.Client
	Repos *domain.Repositories
}

// Initialize 构建各类依赖并返回关闭函数。
func Initialize(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*Container, func(context.Context) error, error) {
	container := &Container{}

	db, err := database.New(ctx, cfg.Database, logger)
	if err != nil {
		return nil, nil, err
	}
	container.DB = db

	dialect := database.NewDialect(cfg.Database.Driver)
	container.Repos = repository.NewSQLRepositories(db, dialect)

	redisClient, err := cache.New(ctx, cfg.Redis, logger)
	if err != nil {
		_ = db.Close()
		return nil, nil, err
	}
	container.Redis = redisClient

	cleanup := func(ctx context.Context) error {
		var errs error
		if container.DB != nil {
			if err := container.DB.Close(); err != nil {
				errs = multierr.Append(errs, err)
			}
		}
		if container.Redis != nil {
			if err := container.Redis.Close(); err != nil {
				errs = multierr.Append(errs, err)
			}
		}
		return errs
	}

	return container, cleanup, nil
}
