package database

import (
	"os"
	"strconv"
	"testing"

	"rainchanel.com/internal/config"
)

func getTestDBConfig() config.DatabaseConfig {
	host := os.Getenv("TEST_DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := 3306
	if portStr := os.Getenv("TEST_DB_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}
	user := os.Getenv("TEST_DB_USER")
	if user == "" {
		user = "root"
	}
	password := os.Getenv("TEST_DB_PASSWORD")
	databaseName := os.Getenv("TEST_DB_NAME")
	if databaseName == "" {
		databaseName = "rainchanel_test"
	}

	return config.DatabaseConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Database: databaseName,
	}
}

func TestInit(t *testing.T) {
	config := getTestDBConfig()
	err := Init(config)
	if err != nil {
		t.Skipf("Skipping test - could not connect to MySQL: %v", err)
	}
	defer Close()

	if DB == nil {
		t.Error("DB should not be nil after Init()")
	}

	var userCount int64
	if err := DB.Model(&User{}).Count(&userCount).Error; err != nil {
		t.Errorf("Failed to query users table: %v", err)
	}

}

func TestInit_InvalidConfig(t *testing.T) {
	invalidConfig := config.DatabaseConfig{
		Host:     "invalid-host",
		Port:     3306,
		User:     "invalid-user",
		Password: "invalid-password",
		Database: "invalid-database",
	}
	err := Init(invalidConfig)
	if err == nil {
		t.Error("Init() should fail with invalid config")
	}
}

func TestInit_ExistingDB(t *testing.T) {
	config := getTestDBConfig()

	err := Init(config)
	if err != nil {
		t.Skipf("Skipping test - could not connect to MySQL: %v", err)
	}

	user := User{
		Username: "testuser",
		Password: "hashedpassword",
	}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	Close()

	err = Init(config)
	if err != nil {
		t.Fatalf("Init() error on reinit = %v", err)
	}

	var foundUser User
	if err := DB.Where("username = ?", "testuser").First(&foundUser).Error; err != nil {
		t.Errorf("User should still exist after reinit: %v", err)
	}

	Close()
}

func TestClose(t *testing.T) {
	config := getTestDBConfig()

	err := Init(config)
	if err != nil {
		t.Skipf("Skipping test - could not connect to MySQL: %v", err)
	}

	err = Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	err = Close()
	if err != nil {
		t.Errorf("Close() error on second call = %v", err)
	}
}

func TestClose_WithoutInit(t *testing.T) {
	DB = nil

	err := Close()
	if err != nil {
		t.Errorf("Close() should not error when DB is nil, got %v", err)
	}
}
