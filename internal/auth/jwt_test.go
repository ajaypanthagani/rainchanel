package auth

import (
	"testing"
	"time"

	"rainchanel.com/internal/config"
)

func TestGenerateToken(t *testing.T) {
	config.App = &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key",
		},
	}

	userID := uint(1)
	username := "testuser"

	token, err := GenerateToken(userID, username)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateToken() returned empty token")
	}

	time.Sleep(5 * time.Millisecond)
	token2, err := GenerateToken(userID, username)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if token == token2 {
		_, err1 := ValidateToken(token)
		_, err2 := ValidateToken(token2)
		if err1 != nil || err2 != nil {
			t.Error("Generated tokens should be valid even if identical")
		}
	}
}

func TestValidateToken(t *testing.T) {
	config.App = &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key",
		},
	}

	userID := uint(1)
	username := "testuser"

	token, err := GenerateToken(userID, username)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("ValidateToken() claims.UserID = %d, want %d", claims.UserID, userID)
	}

	if claims.Username != username {
		t.Errorf("ValidateToken() claims.Username = %s, want %s", claims.Username, username)
	}

	if claims.ExpiresAt == nil {
		t.Error("ValidateToken() claims.ExpiresAt should be set")
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	config.App = &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key",
		},
	}

	testCases := []struct {
		name  string
		token string
	}{
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "invalid format",
			token: "not.a.valid.token",
		},
		{
			name:  "random string",
			token: "random-string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateToken(tc.token)
			if err == nil {
				t.Error("ValidateToken() should return error for invalid token")
			}
		})
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	config.App = &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key",
		},
	}

	userID := uint(1)
	username := "testuser"

	token, err := GenerateToken(userID, username)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	_, err = ValidateToken(token)
	if err != nil {
		t.Errorf("ValidateToken() should succeed with correct secret: %v", err)
	}

	invalidToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJ1c2VybmFtZSI6InRlc3R1c2VyIn0.invalid-signature"
	_, err = ValidateToken(invalidToken)
	if err == nil {
		t.Error("ValidateToken() should return error for token with invalid signature")
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	config.App = &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key",
		},
	}

	userID := uint(1)
	username := "testuser"

	token, err := GenerateToken(userID, username)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if claims.ExpiresAt != nil {
		expTime := claims.ExpiresAt.Time
		if expTime.Before(time.Now()) {
			t.Error("Token expiration should be in the future")
		}

		expectedExp := time.Now().Add(24 * time.Hour)
		diff := expectedExp.Sub(expTime)
		if diff < 0 {
			diff = -diff
		}
		if diff > 1*time.Minute {
			t.Errorf("Token expiration should be approximately 24 hours, got %v", expTime)
		}
	}
}

func TestGenerateToken_DifferentUsers(t *testing.T) {
	config.App = &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key",
		},
	}

	token1, err := GenerateToken(1, "user1")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	token2, err := GenerateToken(2, "user2")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if token1 == token2 {
		t.Error("Tokens for different users should be different")
	}

	claims1, err := ValidateToken(token1)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	claims2, err := ValidateToken(token2)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if claims1.UserID == claims2.UserID {
		t.Error("Claims should have different user IDs")
	}

	if claims1.Username == claims2.Username {
		t.Error("Claims should have different usernames")
	}
}
