package service

import (
	"os"
	"strconv"
	"testing"

	"rainchanel.com/internal/auth"
	"rainchanel.com/internal/config"
	"rainchanel.com/internal/database"
)

func setupTestDB(t *testing.T) {
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

	if err := database.Init(config.DatabaseConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Database: databaseName,
	}); err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	config.App = &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key",
		},
	}
}

func cleanupTestDB(t *testing.T) {
	database.Close()
}

func TestNewAuthService(t *testing.T) {
	service := NewAuthService()
	if service == nil {
		t.Error("NewAuthService() returned nil")
	}
}

func TestAuthService_Register(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	service := NewAuthService()

	tests := []struct {
		name     string
		username string
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "success",
			username: "testuser",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "duplicate username",
			username: "testuser",
			password: "password456",
			wantErr:  true,
			errMsg:   "username already exists",
		},
		{
			name:     "empty username",
			username: "",
			password: "password123",
			wantErr:  false, // Database might allow empty username
		},
		{
			name:     "empty password",
			username: "user2",
			password: "",
			wantErr:  false, // bcrypt allows empty password
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.Register(tt.username, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if err.Error() != tt.errMsg {
					t.Errorf("Register() error message = %v, want %v", err.Error(), tt.errMsg)
				}
			}

			if !tt.wantErr {
				var user database.User
				if err := database.DB.Where("username = ?", tt.username).First(&user).Error; err != nil {
					t.Errorf("User was not created: %v", err)
				}

				if user.Password == tt.password {
					t.Error("Password should be hashed, not stored in plain text")
				}

				if !auth.CheckPasswordHash(tt.password, user.Password) {
					t.Error("Stored password hash should match original password")
				}
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	service := NewAuthService()

	username := "testuser"
	password := "password123"
	if err := service.Register(username, password); err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	tests := []struct {
		name     string
		username string
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "success",
			username: username,
			password: password,
			wantErr:  false,
		},
		{
			name:     "wrong password",
			username: username,
			password: "wrong-password",
			wantErr:  true,
			errMsg:   "invalid username or password",
		},
		{
			name:     "non-existent user",
			username: "nonexistent",
			password: password,
			wantErr:  true,
			errMsg:   "invalid username or password",
		},
		{
			name:     "empty username",
			username: "",
			password: password,
			wantErr:  true,
			errMsg:   "invalid username or password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, userID, returnedUsername, err := service.Login(tt.username, tt.password)

			if (err != nil) != tt.wantErr {
				t.Errorf("Login() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err.Error() != tt.errMsg {
					t.Errorf("Login() error message = %v, want %v", err.Error(), tt.errMsg)
				}
				if token != "" {
					t.Error("Login() should not return token on error")
				}
			} else {
				if token == "" {
					t.Error("Login() should return a token")
				}

				if userID == 0 {
					t.Error("Login() should return a non-zero user ID")
				}

				if returnedUsername != tt.username {
					t.Errorf("Login() returned username = %v, want %v", returnedUsername, tt.username)
				}

				claims, err := auth.ValidateToken(token)
				if err != nil {
					t.Errorf("Login() returned invalid token: %v", err)
				} else {
					if claims.UserID != userID {
						t.Errorf("Token claims.UserID = %d, want %d", claims.UserID, userID)
					}
					if claims.Username != username {
						t.Errorf("Token claims.Username = %s, want %s", claims.Username, username)
					}
				}
			}
		})
	}
}

func TestAuthService_Login_MultipleUsers(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	service := NewAuthService()

	users := []struct {
		username string
		password string
	}{
		{"user1", "pass1"},
		{"user2", "pass2"},
		{"user3", "pass3"},
	}

	for _, u := range users {
		if err := service.Register(u.username, u.password); err != nil {
			t.Fatalf("Failed to register user %s: %v", u.username, err)
		}
	}

	for _, u := range users {
		token, userID, username, err := service.Login(u.username, u.password)
		if err != nil {
			t.Errorf("Login() failed for user %s: %v", u.username, err)
			continue
		}

		if token == "" {
			t.Errorf("Login() returned empty token for user %s", u.username)
		}

		if userID == 0 {
			t.Errorf("Login() returned zero user ID for user %s", u.username)
		}

		if username != u.username {
			t.Errorf("Login() returned username = %s, want %s", username, u.username)
		}

		claims, err := auth.ValidateToken(token)
		if err != nil {
			t.Errorf("Token validation failed for user %s: %v", u.username, err)
		} else {
			if claims.Username != u.username {
				t.Errorf("Token claims username = %s, want %s", claims.Username, u.username)
			}
		}
	}
}
