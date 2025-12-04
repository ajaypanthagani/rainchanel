package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "test-password-123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == "" {
		t.Error("HashPassword() returned empty hash")
	}

	if hash == password {
		t.Error("HashPassword() returned the same value as input")
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == hash2 {
		t.Error("HashPassword() should produce different hashes each time")
	}
}

func TestCheckPasswordHash(t *testing.T) {
	password := "test-password-123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if !CheckPasswordHash(password, hash) {
		t.Error("CheckPasswordHash() should return true for correct password")
	}

	if CheckPasswordHash("wrong-password", hash) {
		t.Error("CheckPasswordHash() should return false for wrong password")
	}

	if CheckPasswordHash("", hash) {
		t.Error("CheckPasswordHash() should return false for empty password")
	}
}

func TestHashPassword_EmptyPassword(t *testing.T) {
	hash, err := HashPassword("")
	if err != nil {
		t.Fatalf("HashPassword() should not error on empty password, got %v", err)
	}

	if hash == "" {
		t.Error("HashPassword() should return a hash even for empty password")
	}

	if !CheckPasswordHash("", hash) {
		t.Error("CheckPasswordHash() should return true for empty password with its hash")
	}
}

func TestCheckPasswordHash_InvalidHash(t *testing.T) {
	invalidHash := "not-a-valid-bcrypt-hash"
	password := "test-password"

	if CheckPasswordHash(password, invalidHash) {
		t.Error("CheckPasswordHash() should return false for invalid hash")
	}
}
