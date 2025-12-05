package service

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
	"rainchanel.com/internal/auth"
	"rainchanel.com/internal/database"
	"rainchanel.com/internal/repository"
)

type AuthService interface {
	Register(username, password string) error
	Login(username, password string) (string, uint, string, error)
}

type authService struct {
	userRepo repository.UserRepository
}

func NewAuthService() AuthService {
	return &authService{
		userRepo: repository.NewUserRepository(),
	}
}

func NewAuthServiceWithRepo(userRepo repository.UserRepository) AuthService {
	return &authService{
		userRepo: userRepo,
	}
}

func (s *authService) Register(username, password string) error {
	_, err := s.userRepo.FindByUsername(username)
	if err == nil {
		return errors.New("username already exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user := database.User{
		Username: username,
		Password: hashedPassword,
	}

	if err := s.userRepo.Create(&user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (s *authService) Login(username, password string) (string, uint, string, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", 0, "", errors.New("invalid username or password")
		}
		return "", 0, "", fmt.Errorf("failed to find user: %w", err)
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
