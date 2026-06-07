package database

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/liangshanbo223/github-demo-project/database/model"
)

func TestInitDB(t *testing.T) {
	// Create a temporary directory for test database
	tempDir, err := os.MkdirTemp("", "s-ui-db-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "s-ui-test.db")

	// 1. Test first-time initialization
	err = InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed on first run: %v", err)
	}

	testDb := GetDB()
	if testDb == nil {
		t.Fatal("GetDB returned nil after initialization")
	}

	// 2. Validate default user creation
	var userCount int64
	err = testDb.Model(&model.User{}).Count(&userCount).Error
	if err != nil {
		t.Fatalf("Failed to count users: %v", err)
	}
	if userCount != 1 {
		t.Errorf("Expected 1 default user, got %d", userCount)
	}

	var defaultUser model.User
	err = testDb.Where("username = ?", "admin").First(&defaultUser).Error
	if err != nil {
		t.Errorf("Default 'admin' user not found: %v", err)
	}

	// 3. Validate master node (ID=0) initialization
	var node model.Node
	err = testDb.Where("id = ?", 0).First(&node).Error
	if err != nil {
		t.Errorf("Default master node (ID=0) not found: %v", err)
	}
	if node.Name != "本地主控" {
		t.Errorf("Expected master node name '本地主控', got '%s'", node.Name)
	}

	// 4. Validate default Reality TLS certificate injection
	var tls model.Tls
	err = testDb.Where("id = ?", 1).First(&tls).Error
	if err != nil {
		t.Errorf("Default Reality certificate (ID=1) not found: %v", err)
	}
	if tls.Name != "默认 Reality 证书" {
		t.Errorf("Expected default TLS name '默认 Reality 证书', got '%s'", tls.Name)
	}

	// 5. Test re-initialization on existing database (idempotency)
	err = InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed on second run: %v", err)
	}

	// Ensure counts haven't duplicated
	err = testDb.Model(&model.User{}).Count(&userCount).Error
	if err != nil {
		t.Fatalf("Failed to count users: %v", err)
	}
	if userCount != 1 {
		t.Errorf("Expected user count to stay 1, got %d", userCount)
	}
}
