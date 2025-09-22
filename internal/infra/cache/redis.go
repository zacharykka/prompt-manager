package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zacharykka/prompt-manager/internal/config"
	"go.uber.org/zap"
)

// New 构建 Redis 客户端并验证连通性。
func New(ctx context.Context, cfg config.RedisConfig, logger *zap.Logger) (*redis.Client, error) {
	options := &redis.Options{
		Addr:     cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	}
	client := redis.NewClient(options)

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	logger.Info("redis connected", zap.String("addr", cfg.Addr))
	return client, nil
}

// Health 检查 Redis 连通性。
func Health(ctx context.Context, client *redis.Client) error {
	if client == nil {
		return errors.New("redis not initialized")
	}
	healthCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return client.Ping(healthCtx).Err()
}
