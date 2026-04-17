package main

// @title           FlowX API
// @version         1.0
// @description     企业级智能工具治理与流程编排平台 API 文档
// @termsOfService  https://github.com/jiangfire/flowx

// @contact.name   jiangfire
// @contact.url    https://github.com/jiangfire/flowx
// @contact.email  neolidy@outlook.com

// @license.name  MIT
// @license.url   https://github.com/jiangfire/flowx/blob/main/LICENSE

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT Bearer Token，格式: Bearer {token}

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.neolidy.top/neo/flowx/internal/app"
	"git.neolidy.top/neo/flowx/internal/application/ai"
	"git.neolidy.top/neo/flowx/internal/config"
	"git.neolidy.top/neo/flowx/internal/infrastructure/cache"
	"git.neolidy.top/neo/flowx/internal/infrastructure/persistence"
	"git.neolidy.top/neo/flowx/internal/infrastructure/server"
	httpInterface "git.neolidy.top/neo/flowx/internal/interfaces/http"

	_ "git.neolidy.top/neo/flowx/docs" // Swagger 文档
)

func main() {
	// 命令行参数
	configPath := flag.String("config", "", "配置文件路径")
	migrateOnly := flag.Bool("migrate", false, "仅执行数据库迁移")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	// 初始化结构化日志
	initLogger(cfg.Log.Level)

	slog.Info("FlowX 服务启动中...")

	// 初始化数据库（传入日志级别）
	db, err := persistence.InitDB(cfg.Database, cfg.Log.Level)
	if err != nil {
		slog.Error("初始化数据库失败", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := persistence.CloseDB(db); err != nil {
			slog.Error("关闭数据库连接失败", "error", err)
		}
	}()

	// 如果仅执行迁移，则退出
	if *migrateOnly {
		slog.Info("数据库迁移完成")
		os.Exit(0)
	}

	// 初始化Redis
	redisClient, err := cache.InitRedis(cfg.Redis)
	if err != nil {
		slog.Error("初始化Redis失败", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := cache.CloseRedis(redisClient); err != nil {
			slog.Error("关闭Redis连接失败", "error", err)
		}
	}()

	// 初始化 LLM 服务
	llmTimeout := time.Duration(cfg.LLM.Timeout) * time.Second
	if llmTimeout <= 0 {
		llmTimeout = 30 * time.Second
	}
	llmSvc := ai.NewLLMService(cfg.LLM.Endpoint, cfg.LLM.APIKey, llmTimeout)

	// 创建Gin引擎
	r := server.NewServer(cfg.Server)

	// 创建服务容器并注册路由
	container := app.NewContainer(db, *cfg, llmSvc)
	httpInterface.SetupRouter(r, container)

	// 创建HTTP Server
	srv := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 在goroutine中启动服务
	go func() {
		slog.Info("HTTP服务已启动",
			"addr", cfg.Server.Addr(),
			"mode", cfg.Server.Mode,
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP服务异常退出", "error", err)
			os.Exit(1)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("收到关闭信号，开始优雅关闭...", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("服务关闭失败", "error", err)
		os.Exit(1)
	}

	slog.Info("FlowX 服务已停止")
}

// initLogger 初始化结构化日志
func initLogger(level string) {
	var slogLevel slog.Level
	switch level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLevel,
	})
	slog.SetDefault(slog.New(handler))
}
