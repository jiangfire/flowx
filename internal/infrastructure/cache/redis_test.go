package cache

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jiangfire/flowx/internal/config"
)

const (
	envRedisHost = "TEST_REDIS_HOST"
	envRedisPort = "TEST_REDIS_PORT"
)

func getTestRedisConfig(t *testing.T) config.RedisConfig {
	t.Helper()
	host := os.Getenv(envRedisHost)
	if host == "" {
		t.Skipf("%s not set, skipping Redis integration test", envRedisHost)
	}
	portStr := os.Getenv(envRedisPort)
	port, _ := strconv.Atoi(portStr)
	if port == 0 {
		port = 6379
	}
	return config.RedisConfig{
		Host: host,
		Port: port,
	}
}

func TestInitRedis_Success(t *testing.T) {
	cfg := getTestRedisConfig(t)
	client, err := InitRedis(cfg)
	if err != nil {
		t.Fatalf("InitRedis failed: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping after init failed: %v", err)
	}

	if err := CloseRedis(client); err != nil {
		t.Fatalf("CloseRedis failed: %v", err)
	}
}

func TestInitRedis_Failure(t *testing.T) {
	cfg := config.RedisConfig{
		Host: "255.255.255.255",
		Port: 6379,
	}
	client, err := InitRedis(cfg)
	if err == nil {
		if client != nil {
			CloseRedis(client)
		}
		t.Fatal("expected InitRedis to fail with invalid address")
	}
}

func TestCloseRedis_NilClient(t *testing.T) {
	if err := CloseRedis(nil); err != nil {
		t.Fatalf("expected nil on nil client, got: %v", err)
	}
}
