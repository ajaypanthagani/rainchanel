package repository

import (
	"errors"

	"gorm.io/gorm"
	"rainchanel.com/internal/database"
)

type UserRepository interface {
	FindByUsername(username string) (*database.User, error)
	Create(user *database.User) error
}

type userRepository struct{}

func NewUserRepository() UserRepository {
	return &userRepository{}
}

func (r *userRepository) FindByUsername(username string) (*database.User, error) {
	if database.DB == nil {
		return nil, errors.New("database not initialized")
	}
	var user database.User
	err := database.DB.Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) Create(user *database.User) error {
	if database.DB == nil {
		return errors.New("database not initialized")
	}
	return database.DB.Create(user).Error
}

