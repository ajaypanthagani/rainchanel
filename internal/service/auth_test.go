package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"rainchanel.com/internal/auth"
	"rainchanel.com/internal/config"
	"rainchanel.com/internal/database"
)

func TestNewAuthService(t *testing.T) {
	service := NewAuthService()
	if service == nil {
		t.Error("NewAuthService() returned nil")
	}
}

func TestAuthService_Register(t *testing.T) {

	config.App = &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key",
		},
	}

	tests := []struct {
		name       string
		username   string
		password   string
		wantErr    bool
		errMsg     string
		setupMocks func() *MockUserRepository
	}{
		{
			name:     "success",
			username: "testuser",
			password: "password123",
			wantErr:  false,
			setupMocks: func() *MockUserRepository {
				return &MockUserRepository{
					FindByUsernameFunc: func(username string) (*database.User, error) {
						return nil, gorm.ErrRecordNotFound
					},
					CreateFunc: func(user *database.User) error {
						user.ID = 1
						return nil
					},
				}
			},
		},
		{
			name:     "duplicate username",
			username: "testuser",
			password: "password456",
			wantErr:  true,
			errMsg:   "username already exists",
			setupMocks: func() *MockUserRepository {
				return &MockUserRepository{
					FindByUsernameFunc: func(username string) (*database.User, error) {
						return &database.User{ID: 1, Username: username}, nil
					},
				}
			},
		},
		{
			name:     "empty username",
			username: "",
			password: "password123",
			wantErr:  false,
			setupMocks: func() *MockUserRepository {
				return &MockUserRepository{
					FindByUsernameFunc: func(username string) (*database.User, error) {
						return nil, gorm.ErrRecordNotFound
					},
					CreateFunc: func(user *database.User) error {
						user.ID = 1
						return nil
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := tt.setupMocks()
			service := NewAuthServiceWithRepo(userRepo)

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
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {

	config.App = &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key",
		},
	}

	username := "testuser"
	password := "password123"
	hashedPassword, _ := auth.HashPassword(password)

	tests := []struct {
		name       string
		username   string
		password   string
		wantErr    bool
		errMsg     string
		setupMocks func() *MockUserRepository
	}{
		{
			name:     "success",
			username: username,
			password: password,
			wantErr:  false,
			setupMocks: func() *MockUserRepository {
				return &MockUserRepository{
					FindByUsernameFunc: func(u string) (*database.User, error) {
						return &database.User{
							ID:       1,
							Username: username,
							Password: hashedPassword,
						}, nil
					},
				}
			},
		},
		{
			name:     "wrong password",
			username: username,
			password: "wrong-password",
			wantErr:  true,
			errMsg:   "invalid username or password",
			setupMocks: func() *MockUserRepository {
				return &MockUserRepository{
					FindByUsernameFunc: func(u string) (*database.User, error) {
						return &database.User{
							ID:       1,
							Username: username,
							Password: hashedPassword,
						}, nil
					},
				}
			},
		},
		{
			name:     "non-existent user",
			username: "nonexistent",
			password: password,
			wantErr:  true,
			errMsg:   "invalid username or password",
			setupMocks: func() *MockUserRepository {
				return &MockUserRepository{
					FindByUsernameFunc: func(u string) (*database.User, error) {
						return nil, gorm.ErrRecordNotFound
					},
				}
			},
		},
		{
			name:     "empty username",
			username: "",
			password: password,
			wantErr:  true,
			errMsg:   "invalid username or password",
			setupMocks: func() *MockUserRepository {
				return &MockUserRepository{
					FindByUsernameFunc: func(u string) (*database.User, error) {
						return nil, gorm.ErrRecordNotFound
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := tt.setupMocks()
			service := NewAuthServiceWithRepo(userRepo)

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

	config.App = &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key",
		},
	}

	users := []struct {
		username string
		password string
		userID   uint
	}{
		{"user1", "pass1", 1},
		{"user2", "pass2", 2},
		{"user3", "pass3", 3},
	}

	userMap := make(map[string]*database.User)
	for _, u := range users {
		hashedPassword, _ := auth.HashPassword(u.password)
		userMap[u.username] = &database.User{
			ID:       u.userID,
			Username: u.username,
			Password: hashedPassword,
		}
	}

	userRepo := &MockUserRepository{
		FindByUsernameFunc: func(username string) (*database.User, error) {
			user, exists := userMap[username]
			if !exists {
				return nil, gorm.ErrRecordNotFound
			}
			return user, nil
		},
		CreateFunc: func(user *database.User) error {
			userMap[user.Username] = user
			return nil
		},
	}

	service := NewAuthServiceWithRepo(userRepo)

	for _, u := range users {
		user := &database.User{
			Username: u.username,
		}
		hashedPassword, _ := auth.HashPassword(u.password)
		user.Password = hashedPassword
		user.ID = u.userID
		userMap[u.username] = user
	}

	for _, u := range users {
		t.Run("login_"+u.username, func(t *testing.T) {
			token, userID, username, err := service.Login(u.username, u.password)
			if err != nil {
				t.Errorf("Login() failed for user %s: %v", u.username, err)
				return
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
		})
	}
}
