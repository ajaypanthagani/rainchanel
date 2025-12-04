package service

import (
	"errors"
	"fmt"

	"rainchanel.com/internal/auth"
	"rainchanel.com/internal/database"
)

type AuthService interface {
	Register(username, password string) error
	Login(username, password string) (string, uint, string, error)
}

type authService struct{}

func NewAuthService() AuthService {
	return &authService{}
}

func (s *authService) Register(username, password string) error {
	var existingUser database.User
	if err := database.DB.Where("username = ?", username).First(&existingUser).Error; err == nil {
		return errors.New("username already exists")
	}

	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user := database.User{
		Username: username,
		Password: hashedPassword,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (s *authService) Login(username, password string) (string, uint, string, error) {
	var user database.User
	if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return "", 0, "", errors.New("invalid username or password")
	}

	if !auth.CheckPasswordHash(password, user.Password) {
		return "", 0, "", errors.New("invalid username or password")
	}

	token, err := auth.GenerateToken(user.ID, user.Username)
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return token, user.ID, user.Username, nil
}
