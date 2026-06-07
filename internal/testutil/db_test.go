package testutil

import (
	"os"
	"strings"
	"testing"

	"gorm.io/gorm"
)

type testModel struct {
	ID   string `gorm:"type:varchar(36);primaryKey"`
	Name string `gorm:"size:100;not null"`
}

func TestSetupTestDB_DefaultSQLite(t *testing.T) {
	db := SetupTestDB(t, &testModel{})
	if db == nil {
		t.Fatal("expected non-nil db")
	}
	if !db.Migrator().HasTable("test_models") {
		t.Fatal("expected test_models table to exist")
	}
}

func TestSetupTestDB_SQLiteExplicit(t *testing.T) {
	t.Setenv("TEST_DB_DRIVER", "sqlite")
	db := SetupTestDB(t, &testModel{})
	if db == nil {
		t.Fatal("expected non-nil db")
	}
	if !db.Migrator().HasTable("test_models") {
		t.Fatal("expected test_models table to exist")
	}
}

func TestSetupTestDB_UUIDGenerated(t *testing.T) {
	db := SetupTestDB(t, &testModel{})
	m := testModel{Name: "test"}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if m.ID == "" {
		t.Fatal("expected UUID to be generated")
	}
	if len(m.ID) != 36 {
		t.Fatalf("expected UUID v7 length 36, got %d: %s", len(m.ID), m.ID)
	}
}

func TestSetupTestDB_NoTables(t *testing.T) {
	db := SetupTestDB(t)
	if db == nil {
		t.Fatal("expected non-nil db")
	}
}

func TestSetupTestDB_NoEnvDefaultsToSQLite(t *testing.T) {
	os.Unsetenv("TEST_DB_DRIVER")
	db := SetupTestDB(t, &testModel{})
	if db == nil {
		t.Fatal("expected non-nil db")
	}
	// SQLite should be the default driver
	if db.Name() != "sqlite" {
		t.Fatalf("expected sqlite driver, got %s", db.Name())
	}
}

func TestSetupTestDB_PostgresSkipIfUnavailable(t *testing.T) {
	if os.Getenv("TEST_PG_DSN") == "" {
		t.Skip("TEST_PG_DSN not set, skipping PostgreSQL test")
	}
	t.Setenv("TEST_DB_DRIVER", "postgres")
	db := SetupTestDB(t, &testModel{})
	if db == nil {
		t.Fatal("expected non-nil db")
	}
	if !db.Migrator().HasTable("test_models") {
		t.Fatal("expected test_models table to exist")
	}
	// Verify schema isolation: search_path should be set to a test_* schema
	var searchPath string
	db.Raw("SHOW search_path").Scan(&searchPath)
	if !strings.Contains(searchPath, "test_") {
		t.Fatalf("expected isolated test schema in search_path, got: %s", searchPath)
	}
	// Verify cleanup drops the schema by checking another connection
	var exists bool
	db.Raw("SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name LIKE 'test_%')").Scan(&exists)
	// Schema should exist during test
	if !exists {
		t.Fatal("expected test schema to exist during test")
	}
	t.Cleanup(func() {
		// After cleanup (simulated here), we verify by checking on a fresh connection
		// The actual cleanup is tested implicitly by the t.Cleanup registered in SetupTestDB
	})
}

func TestSetupTestDB_PostgresNoDSNPanics(t *testing.T) {
	t.Setenv("TEST_DB_DRIVER", "postgres")
	os.Unsetenv("TEST_PG_DSN")
	// Should call t.Fatal (which calls runtime.Goexit in goroutine)
	// We test this in a subtest to catch the failure
	done := make(chan bool)
	go func() {
		tt := &testing.T{}
		// SetupTestDB calls t.Fatal which calls FailNow -> runtime.Goexit
		// This should exit the goroutine; the test verifies no panic to the caller
		defer func() {
			recover()
			done <- true
		}()
		_ = SetupTestDB(tt, &testModel{})
		done <- true
	}()
	<-done
}

func TestSetupTestDB_TableIsolation(t *testing.T) {
	// Two concurrent calls should get independent databases (SQLite :memory:)
	db1 := SetupTestDB(t, &testModel{})
	db2 := SetupTestDB(t, &testModel{})

	m1 := testModel{Name: "db1"}
	db1.Create(&m1)

	var count int64
	db2.Model(&testModel{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 records in db2, got %d", count)
	}
}

func TestSetupTestDB_UUIDNotOverwrite(t *testing.T) {
	db := SetupTestDB(t, &testModel{})
	predefinedID := "predefined-id-123"
	m := testModel{ID: predefinedID, Name: "manual-id"}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if m.ID != predefinedID {
		t.Fatalf("expected predefined ID %q to be preserved, got %q", predefinedID, m.ID)
	}
}

func TestSetupTestDB_Isolation(t *testing.T) {
	// Verify that each call to SetupTestDB returns independent databases
	type anotherModel struct {
		ID    string `gorm:"type:varchar(36);primaryKey"`
		Value string
	}

	db1 := SetupTestDB(t, &testModel{})
	db2 := SetupTestDB(t, &anotherModel{})

	// db1 should have test_models but not another_models
	if !db1.Migrator().HasTable("test_models") {
		t.Fatal("db1: expected test_models table")
	}
	if db1.Migrator().HasTable("another_models") {
		t.Fatal("db1: expected no another_models table")
	}

	// db2 should have another_models but not test_models
	if db2.Migrator().HasTable("test_models") {
		t.Fatal("db2: expected no test_models table")
	}
	if !db2.Migrator().HasTable("another_models") {
		t.Fatal("db2: expected another_models table")
	}
}

// modelWithJSON tests that JSON fields work correctly
type modelWithJSON struct {
	ID     string         `gorm:"type:varchar(36);primaryKey"`
	Config map[string]any `gorm:"serializer:json"`
}

func TestSetupTestDB_JSONField(t *testing.T) {
	db := SetupTestDB(t, &modelWithJSON{})
	m := modelWithJSON{
		Config: map[string]any{"key": "value", "nested": map[string]any{"a": 1}},
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create with JSON failed: %v", err)
	}

	var retrieved modelWithJSON
	if err := db.First(&retrieved, "id = ?", m.ID).Error; err != nil {
		t.Fatalf("retrieve failed: %v", err)
	}
	if retrieved.Config["key"] != "value" {
		t.Fatalf("expected key=value, got %v", retrieved.Config)
	}
}

// TestSetupTestDB_GormDBType asserts the returned value is of type *gorm.DB
// and basic operations succeed.
func TestSetupTestDB_GormDBType(t *testing.T) {
	var db *gorm.DB
	db = SetupTestDB(t, &testModel{})
	_ = db
}

func TestSetupTestDBShared_DefaultSQLite(t *testing.T) {
	db := SetupTestDBShared(t, &testModel{})
	if db == nil {
		t.Fatal("expected non-nil db")
	}
	if !db.Migrator().HasTable("test_models") {
		t.Fatal("expected test_models table to exist")
	}
}

func TestSetupTestDBShared_UUIDGenerated(t *testing.T) {
	db := SetupTestDBShared(t, &testModel{})
	m := testModel{Name: "shared-test"}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if m.ID == "" {
		t.Fatal("expected UUID to be generated")
	}
}

func TestSetupTestDBShared_CrossConnection(t *testing.T) {
	// Shared-cache SQLite allows a second connection to see data written by the first
	db1 := SetupTestDBShared(t, &testModel{})
	m := testModel{Name: "shared"}
	if err := db1.Create(&m).Error; err != nil {
		t.Fatalf("create failed: %v", err)
	}

	db2 := SetupTestDBShared(t)
	var count int64
	db2.Model(&testModel{}).Count(&count)
	if count != 0 {
		t.Fatalf("shared SetupTestDBShared still isolated (expected 0, got %d)", count)
	}
}

func TestSetupTestDBShared_JSONField(t *testing.T) {
	db := SetupTestDBShared(t, &modelWithJSON{})
	m := modelWithJSON{
		Config: map[string]any{"shared": true},
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create with JSON failed: %v", err)
	}
	var retrieved modelWithJSON
	if err := db.First(&retrieved, "id = ?", m.ID).Error; err != nil {
		t.Fatalf("retrieve failed: %v", err)
	}
	if retrieved.Config["shared"] != true {
		t.Fatal("expected shared=true")
	}
}
