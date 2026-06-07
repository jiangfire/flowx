package testutil

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var sharedDBCounter int64

const (
	envDriver = "TEST_DB_DRIVER"
	envPGDSN  = "TEST_PG_DSN"
)

// SetupTestDB creates a test database. Driver: TEST_DB_DRIVER env (default "sqlite").
// For PostgreSQL, TEST_PG_DSN must be set; each call creates an isolated schema
// dropped in t.Cleanup. SQLite uses ":memory:" (isolated per call). Tables are
// auto-migrated.
func SetupTestDB(t *testing.T, tables ...interface{}) *gorm.DB {
	return setupDB(t, false, tables...)
}

// SetupTestDBShared is like SetupTestDB but uses shared-cache SQLite
// (file:mem_N?mode=memory&cache=shared) for transaction tests where
// DBFromContext picks up a different *gorm.DB inside WithTransaction.
func SetupTestDBShared(t *testing.T, tables ...interface{}) *gorm.DB {
	return setupDB(t, true, tables...)
}

func setupDB(t *testing.T, shared bool, tables ...interface{}) *gorm.DB {
	t.Helper()

	driver := strings.ToLower(os.Getenv(envDriver))
	if driver == "" {
		driver = "sqlite"
	}

	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	var db *gorm.DB
	var err error

	switch driver {
	case "postgres", "pg", "postgresql":
		dsn := os.Getenv(envPGDSN)
		if dsn == "" {
			t.Fatalf("TEST_PG_DSN env not set (required when TEST_DB_DRIVER=%s)", driver)
		}
		db, err = gorm.Open(gormpostgres.Open(dsn), gormCfg)
		if err != nil {
			t.Fatalf("connect PostgreSQL failed: %v", err)
		}
		schema := fmt.Sprintf("test_%s", strings.ReplaceAll(uuid.NewString(), "-", "_"))
		if err := db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema)).Error; err != nil {
			t.Fatalf("create schema %s failed: %v", schema, err)
		}
		if err := db.Exec(fmt.Sprintf("SET search_path TO %s", schema)).Error; err != nil {
			t.Fatalf("set search_path failed: %v", err)
		}
		t.Cleanup(func() {
			db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema))
			sqlDB, _ := db.DB()
			if sqlDB != nil {
				sqlDB.Close()
			}
		})
	default:
		dsn := ":memory:"
		if shared {
			dsn = fmt.Sprintf("file:mem_%d?mode=memory&cache=shared", atomic.AddInt64(&sharedDBCounter, 1))
		}
		db, err = gorm.Open(sqlite.Open(dsn), gormCfg)
		if err != nil {
			t.Fatalf("create SQLite test db failed: %v", err)
		}
		t.Cleanup(func() {
			sqlDB, _ := db.DB()
			if sqlDB != nil {
				sqlDB.Close()
			}
		})
	}

	registerUUIDCallback(db)

	if len(tables) > 0 {
		if err := db.AutoMigrate(tables...); err != nil {
			t.Fatalf("auto migrate failed: %v", err)
		}
	}

	return db
}

func registerUUIDCallback(db *gorm.DB) {
	_ = db.Callback().Create().Before("gorm:create").Register("uuid_v7_generator", func(db *gorm.DB) {
		if db.Statement.Schema == nil {
			return
		}
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
	})
}
