package infra

import (
	"context"
	"database/sql"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zacharykka/prompt-manager/internal/config"
	"github.com/zacharykka/prompt-manager/internal/domain"
	"github.com/zacharykka/prompt-manager/internal/infra/cache"
	"github.com/zacharykka/prompt-manager/internal/infra/database"
	"github.com/zacharykka/prompt-manager/internal/infra/repository"
	"github.com/zacharykka/prompt-manager/internal/middleware"
	authutil "github.com/zacharykka/prompt-manager/pkg/auth"
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

	if err := ensureDefaultAdmin(ctx, cfg, container.Repos, logger); err != nil {
		_ = db.Close()
		_ = redisClient.Close()
		return nil, nil, err
	}

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

func ensureDefaultAdmin(ctx context.Context, cfg *config.Config, repos *domain.Repositories, logger *zap.Logger) error {
	email := strings.ToLower(strings.TrimSpace(cfg.Seed.Admin.Email))
	password := cfg.Seed.Admin.Password
	role := strings.ToLower(strings.TrimSpace(cfg.Seed.Admin.Role))

	// 向后兼容旧环境变量
	legacyEmail := strings.ToLower(strings.TrimSpace(os.Getenv("PROMPT_MANAGER_INIT_ADMIN_EMAIL")))
	legacyPassword := os.Getenv("PROMPT_MANAGER_INIT_ADMIN_PASSWORD")
	legacyRole := strings.ToLower(strings.TrimSpace(os.Getenv("PROMPT_MANAGER_INIT_ADMIN_ROLE")))

	if email == "" {
		email = legacyEmail
	}
	if password == "" {
		password = legacyPassword
	}
	if role == "" {
		role = legacyRole
	}

	if email == "" || password == "" {
		logger.Info("admin seeding skipped; seed admin email or password not set")
		return nil
	}

	if _, err := repos.Users.GetByEmail(ctx, email); err == nil {
		logger.Info("seed admin exists", zap.String("email", email))
		return nil
	} else if err != domain.ErrNotFound {
		return err
	}

	if role == "" {
		role = middleware.RoleAdmin
	}

	hash, err := authutil.HashPassword(password)
	if err != nil {
		return err
	}

	admin := &domain.User{
		ID:             uuid.NewString(),
		Email:          email,
		HashedPassword: hash,
		Role:           role,
		Status:         "active",
	}
	if err := repos.Users.Create(ctx, admin); err != nil {
		return err
	}

	logger.Info("seed admin created", zap.String("email", email))
	return nil
}
