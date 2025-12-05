package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"rainchanel.com/internal/api/request"
	"rainchanel.com/internal/api/response"
)

type MockAuthService struct {
	RegisterFunc func(username, password string) error
	LoginFunc    func(username, password string) (string, uint, string, error)
}

func (m *MockAuthService) Register(username, password string) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(username, password)
	}
	return nil
}

func (m *MockAuthService) Login(username, password string) (string, uint, string, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(username, password)
	}
	return "token", 1, "testuser", nil
}

func TestNewAuthHandler(t *testing.T) {
	mockService := &MockAuthService{}
	handler := NewAuthHandler(mockService)

	if handler == nil {
		t.Error("NewAuthHandler() returned nil")
	}
}

func TestAuthHandler_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		serviceError   error
		wantStatusCode int
	}{
		{
			name: "success",
			requestBody: request.RegisterRequest{
				Username: "testuser",
				Password: "password123",
			},
			serviceError:   nil,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			serviceError:   nil,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "missing username",
			requestBody: map[string]interface{}{
				"password": "password123",
			},
			serviceError:   nil,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "missing password",
			requestBody: map[string]interface{}{
				"username": "testuser",
			},
			serviceError:   nil,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "service error - username exists",
			requestBody: request.RegisterRequest{
				Username: "existinguser",
				Password: "password123",
			},
			serviceError:   errors.New("username already exists"),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "short password",
			requestBody: request.RegisterRequest{
				Username: "testuser",
				Password: "123",
			},
			serviceError:   nil,
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAuthService{
				RegisterFunc: func(username, password string) error {
					return tt.serviceError
				},
			}

			handler := NewAuthHandler(mockService)

			router := gin.New()
			router.POST("/register", handler.Register)

			var bodyBytes []byte
			var err error
			if tt.name == "invalid JSON" {
				bodyBytes = []byte("invalid json")
			} else {
				bodyBytes, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantStatusCode == http.StatusOK {
				var resp response.Response
				err = json.Unmarshal(w.Body.Bytes(), &resp)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				registerResp, ok := resp.Data.(map[string]any)
				if !ok {
					t.Error("Response data is not a map")
				} else {
					message, ok := registerResp["message"].(string)
					if !ok {
						t.Error("message is not a string")
					} else if message != "User registered successfully" {
						t.Errorf("message = %v, want 'User registered successfully'", message)
					}
				}
			}
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name            string
		requestBody     interface{}
		serviceToken    string
		serviceUserID   uint
		serviceUsername string
		serviceError    error
		wantStatusCode  int
	}{
		{
			name: "success",
			requestBody: request.LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			serviceToken:    "test-token-123",
			serviceUserID:   1,
			serviceUsername: "testuser",
			serviceError:    nil,
			wantStatusCode:  http.StatusOK,
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			serviceError:   nil,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "missing username",
			requestBody: map[string]interface{}{
				"password": "password123",
			},
			serviceError:   nil,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "missing password",
			requestBody: map[string]interface{}{
				"username": "testuser",
			},
			serviceError:   nil,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "service error - invalid credentials",
			requestBody: request.LoginRequest{
				Username: "testuser",
				Password: "wrongpassword",
			},
			serviceError:   errors.New("invalid username or password"),
			wantStatusCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAuthService{
				LoginFunc: func(username, password string) (string, uint, string, error) {
					return tt.serviceToken, tt.serviceUserID, tt.serviceUsername, tt.serviceError
				},
			}

			handler := NewAuthHandler(mockService)

			router := gin.New()
			router.POST("/login", handler.Login)

			var bodyBytes []byte
			var err error
			if tt.name == "invalid JSON" {
				bodyBytes = []byte("invalid json")
			} else {
				bodyBytes, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantStatusCode == http.StatusOK {
				var resp response.Response
				err = json.Unmarshal(w.Body.Bytes(), &resp)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				loginResp, ok := resp.Data.(map[string]any)
				if !ok {
					t.Error("Response data is not a map")
				} else {
					token, ok := loginResp["token"].(string)
					if !ok {
						t.Error("token is not a string")
					} else if token != tt.serviceToken {
						t.Errorf("token = %v, want %v", token, tt.serviceToken)
					}

					userID, ok := loginResp["user_id"].(float64)
					if !ok {
						t.Error("user_id is not a number")
					} else if uint(userID) != tt.serviceUserID {
						t.Errorf("user_id = %v, want %v", userID, tt.serviceUserID)
					}

					username, ok := loginResp["username"].(string)
					if !ok {
						t.Error("username is not a string")
					} else if username != tt.serviceUsername {
						t.Errorf("username = %v, want %v", username, tt.serviceUsername)
					}
				}
			}
		})
	}
}
