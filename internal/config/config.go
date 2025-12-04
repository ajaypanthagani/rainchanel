package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type JWTConfig struct {
	Secret string `yaml:"secret"`
}

var (
	App *Config
)

func Load() error {
	configPath := "application.yaml"
	App = &Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     3306,
			User:     "root",
			Password: "",
			Database: "rainchanel",
		},
		JWT: JWTConfig{
			Secret: "your-secret-key-change-in-production",
		},
	}

	if _, err := os.Stat(configPath); err == nil {
		if err := loadFromYAML(configPath); err != nil {
			return fmt.Errorf("failed to load config from YAML: %w", err)
		}
		log.Printf("Loaded configuration from %s", configPath)
	} else {
		log.Printf("Config file %s not found, using defaults and environment variables", configPath)
	}

	loadFromEnv()

	return nil
}

func loadFromYAML(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, App); err != nil {
		return err
	}

	return nil
}

func loadFromEnv() {
	if portStr := os.Getenv("SERVER_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			App.Server.Port = port
		}
	}

	if host := os.Getenv("DB_HOST"); host != "" {
		App.Database.Host = host
	}
	if portStr := os.Getenv("DB_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			App.Database.Port = port
		}
	}
	if user := os.Getenv("DB_USER"); user != "" {
		App.Database.User = user
	}
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		App.Database.Password = password
	}
	if database := os.Getenv("DB_NAME"); database != "" {
		App.Database.Database = database
	}
}
