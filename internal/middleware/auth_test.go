package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"rainchanel.com/internal/auth"
	"rainchanel.com/internal/config"
)

func setupMiddlewareTest(t *testing.T) {
	config.App = &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key",
		},
	}
}

func TestAuthMiddleware_NoHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupMiddlewareTest(t)

	router := gin.New()
	router.GET("/protected", AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupMiddlewareTest(t)

	router := gin.New()
	router.GET("/protected", AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	testCases := []struct {
		name       string
		authHeader string
	}{
		{
			name:       "no Bearer prefix",
			authHeader: "token123",
		},
		{
			name:       "missing token",
			authHeader: "Bearer",
		},
		{
			name:       "empty token",
			authHeader: "Bearer ",
		},
		{
			name:       "multiple spaces",
			authHeader: "Bearer token1 token2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", tc.authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupMiddlewareTest(t)

	router := gin.New()
	router.GET("/protected", AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	testCases := []struct {
		name  string
		token string
	}{
		{
			name:  "invalid token format",
			token: "invalid.token.format",
		},
		{
			name:  "random string",
			token: "random-string",
		},
		{
			name:  "empty token",
			token: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+tc.token)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupMiddlewareTest(t)

	userID := uint(1)
	username := "testuser"
	token, err := auth.GenerateToken(userID, username)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	router := gin.New()
	router.GET("/protected", AuthMiddleware(), func(c *gin.Context) {
		ctxUserID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user_id not set"})
			return
		}

		ctxUsername, exists := c.Get("username")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "username not set"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "success",
			"user_id":  ctxUserID,
			"username": ctxUsername,
		})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
		if response["user_id"] != float64(userID) {
			t.Errorf("Expected user_id %d in response, got %v", userID, response["user_id"])
		}
		if response["username"] != username {
			t.Errorf("Expected username %s in response, got %v", username, response["username"])
		}
	}
}

func TestAuthMiddleware_ContextValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupMiddlewareTest(t)

	userID := uint(42)
	username := "contextuser"
	token, err := auth.GenerateToken(userID, username)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	router := gin.New()
	router.GET("/protected", AuthMiddleware(), func(c *gin.Context) {
		ctxUserID, exists := c.Get("user_id")
		assert.True(t, exists, "user_id should exist in context")
		assert.Equal(t, userID, ctxUserID, "user_id should match")

		ctxUsername, exists := c.Get("username")
		assert.True(t, exists, "username should exist in context")
		assert.Equal(t, username, ctxUsername, "username should match")

		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
