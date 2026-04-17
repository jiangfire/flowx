package persistence

import (
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"git.neolidy.top/neo/flowx/internal/config"
	"git.neolidy.top/neo/flowx/internal/domain/agent"
	"git.neolidy.top/neo/flowx/internal/domain/approval"
	"git.neolidy.top/neo/flowx/internal/domain/bpmn"
	"git.neolidy.top/neo/flowx/internal/domain/datagov"
	"git.neolidy.top/neo/flowx/internal/domain/notification"
	"git.neolidy.top/neo/flowx/internal/domain/tenant"
	"git.neolidy.top/neo/flowx/internal/domain/tool"
	"git.neolidy.top/neo/flowx/internal/domain/workflow"

	"github.com/google/uuid"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// InitDB 初始化PostgreSQL数据库连接
func InitDB(cfg config.DatabaseConfig, logLevel string) (*gorm.DB, error) {
	// 根据日志级别映射 GORM 日志级别
	var gormLogLevel gormlogger.LogLevel
	switch strings.ToLower(logLevel) {
	case "silent":
		gormLogLevel = gormlogger.Silent
	case "error":
		gormLogLevel = gormlogger.Error
	case "warn":
		gormLogLevel = gormlogger.Warn
	default:
		gormLogLevel = gormlogger.Info
	}

	db, err := gorm.Open(gormpostgres.Open(cfg.DSN()), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormLogLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 获取底层sql.DB以配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层数据库连接失败: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)
	if cfg.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTime) * time.Second)
	}

	// 注册UUID v7生成器作为默认主键生成回调
	db.Callback().Create().Before("gorm:create").Register("uuid_v7_generator", func(db *gorm.DB) {
		if db.Statement.Schema != nil {
			// 查找名为ID的string类型字段
			for _, field := range db.Statement.Schema.Fields {
				if field.Name == "ID" && field.FieldType.Kind() == reflect.String {
					if db.Statement.ReflectValue.CanAddr() {
						idField := db.Statement.ReflectValue.FieldByName("ID")
						if idField.IsValid() && idField.IsZero() {
							idField.SetString(uuid.Must(uuid.NewV7()).String())
						}
					}
					break
				}
			}
		}
	})

	// 自动迁移所有领域模型
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	slog.Info("数据库连接初始化成功",
		"host", cfg.Host,
		"port", cfg.Port,
		"dbname", cfg.DBName,
	)

	return db, nil
}

// autoMigrate 执行数据库自动迁移
func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&tool.Tool{},
		&tool.Connector{},
		&workflow.Workflow{},
		&approval.Workflow{},
		&approval.WorkflowInstance{},
		&approval.Approval{},
		&datagov.DataPolicy{},
		&datagov.DataAsset{},
		&datagov.DataQualityRule{},
		&datagov.DataQualityCheck{},
		&tenant.Tenant{},
		&tenant.User{},
		&tenant.Role{},
		&tenant.Permission{},
		&agent.AgentTask{},
		&notification.Notification{},
		&notification.NotificationTemplate{},
		&notification.NotificationPreference{},
		&bpmn.ProcessInstance{},
		&bpmn.ProcessTask{},
		&bpmn.ExecutionHistory{},
		&processDefinitionPO{},
	)
}

// CloseDB 关闭数据库连接
func CloseDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
