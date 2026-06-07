package migration

import (
	"os"
	"testing"

	"github.com/admin8800/s-ui/config"
	"github.com/admin8800/s-ui/database/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestMigrateDb_From1_2(t *testing.T) {
	// Setup test environment
	tempDir, err := os.MkdirTemp("", "s-ui-migration-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set SUI_DB_FOLDER environment variable
	t.Setenv("SUI_DB_FOLDER", tempDir)

	dbPath := config.GetDBPath()

	// Initialize database with GORM
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// 1. Pre-create tables simulating a 1.2 database state
	err = db.AutoMigrate(&model.Setting{}, &model.Client{}, &model.Inbound{}, &model.Outbound{})
	if err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	// Insert version settings as 1.2
	err = db.Create(&model.Setting{Key: "version", Value: "1.2.0"}).Error
	if err != nil {
		t.Fatalf("Failed to write mock version: %v", err)
	}

	// Close db to allow MigrateDb() to open it
	sqlDB, _ := db.DB()
	sqlDB.Close()

	// 2. Perform Migration (1.2 -> Latest)
	// This should run without panic/exit
	MigrateDb()

	// 3. Re-open database and verify results
	db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to re-open test database: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	var currentVersion string
	err = db.Model(&model.Setting{}).Select("value").Where("key = ?", "version").First(&currentVersion).Error
	if err != nil {
		t.Fatalf("Failed to retrieve version after migration: %v", err)
	}

	expectedVersion := config.GetVersion()
	if currentVersion != expectedVersion {
		t.Errorf("Expected database version to be migrated to %s, got %s", expectedVersion, currentVersion)
	}
}

func TestMigrateDb_LegacyWithoutConfigKey(t *testing.T) {
	// Setup test environment
	tempDir, err := os.MkdirTemp("", "s-ui-migration-legacy-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Setenv("SUI_DB_FOLDER", tempDir)

	dbPath := config.GetDBPath()

	// Create a database with 1.2 state but NO "config" key in settings (Edge case A)
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	err = db.AutoMigrate(&model.Setting{}, &model.Client{}, &model.Inbound{}, &model.Outbound{})
	if err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	// version is "1.2" but no config setting exists
	err = db.Create(&model.Setting{Key: "version", Value: "1.2.0"}).Error
	if err != nil {
		t.Fatalf("Failed to write mock version: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.Close()

	// Run migration. The missing "config" record should be safely ignored and not log.Fatal/panic
	MigrateDb()

	// Verify database is successfully upgraded to latest
	db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to re-open test database: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	var currentVersion string
	err = db.Model(&model.Setting{}).Select("value").Where("key = ?", "version").First(&currentVersion).Error
	if err != nil {
		t.Fatalf("Failed to retrieve version: %v", err)
	}

	expectedVersion := config.GetVersion()
	if currentVersion != expectedVersion {
		t.Errorf("Expected database version to be migrated to %s, got %s", expectedVersion, currentVersion)
	}
}
