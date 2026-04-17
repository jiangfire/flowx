package cache

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"git.neolidy.top/neo/flowx/internal/config"

	"github.com/redis/go-redis/v9"
)

// InitRedis 初始化Redis连接
func InitRedis(cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Ping测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis连接失败: %w", err)
	}

	slog.Info("Redis连接初始化成功",
		"host", cfg.Host,
		"port", cfg.Port,
		"db", cfg.DB,
	)

	return client, nil
}

// CloseRedis 关闭Redis连接
func CloseRedis(client *redis.Client) error {
	return client.Close()
}
