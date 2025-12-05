package repository

import (
	"errors"

	"gorm.io/gorm"
	"rainchanel.com/internal/database"
)

type ResultRepository interface {
	CreateResult(result *database.Result) error
	FindResultByTaskID(taskID uint) (*database.Result, error)
	FindResultsByUserID(userID uint) ([]database.Result, error)
	FindResultByID(resultID uint) (*database.Result, error)
	FindOldestUnconsumedResultByUserID(userID uint) (*database.Result, error)
	MarkResultAsConsumed(resultID uint) error
}

type resultRepository struct{}

func NewResultRepository() ResultRepository {
	return &resultRepository{}
}

func (r *resultRepository) CreateResult(result *database.Result) error {
	if database.DB == nil {
		return errors.New("database not initialized")
	}
	return database.DB.Create(result).Error
}

func (r *resultRepository) FindResultByTaskID(taskID uint) (*database.Result, error) {
	if database.DB == nil {
		return nil, errors.New("database not initialized")
	}
	var result database.Result
	err := database.DB.Where("task_id = ?", taskID).First(&result).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *resultRepository) FindResultsByUserID(userID uint) ([]database.Result, error) {
	if database.DB == nil {
		return nil, errors.New("database not initialized")
	}
	var results []database.Result
	err := database.DB.Where("created_by = ?", userID).Order("created_at DESC").Find(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (r *resultRepository) FindResultByID(resultID uint) (*database.Result, error) {
	if database.DB == nil {
		return nil, errors.New("database not initialized")
	}
	var result database.Result
	err := database.DB.Where("id = ?", resultID).First(&result).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *resultRepository) FindOldestUnconsumedResultByUserID(userID uint) (*database.Result, error) {
	if database.DB == nil {
		return nil, errors.New("database not initialized")
	}
	var result database.Result
	err := database.DB.Where("created_by = ? AND consumed = ?", userID, false).
		Order("created_at ASC").
		First(&result).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *resultRepository) MarkResultAsConsumed(resultID uint) error {
	if database.DB == nil {
		return errors.New("database not initialized")
	}
	return database.DB.Model(&database.Result{}).
		Where("id = ?", resultID).
		Update("consumed", true).Error
}
