package repository

import (
	"errors"

	"gorm.io/gorm"
	"rainchanel.com/internal/database"
)

type TaskRepository interface {
	CreateTask(task *database.Task) error
	FindTaskByID(taskID uint) (*database.Task, error)
}

type taskRepository struct{}

func NewTaskRepository() TaskRepository {
	return &taskRepository{}
}

func (r *taskRepository) CreateTask(task *database.Task) error {
	if database.DB == nil {
		return errors.New("database not initialized")
	}
	return database.DB.Create(task).Error
}

func (r *taskRepository) FindTaskByID(taskID uint) (*database.Task, error) {
	if database.DB == nil {
		return nil, errors.New("database not initialized")
	}
	var task database.Task
	err := database.DB.Where("id = ?", taskID).First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

