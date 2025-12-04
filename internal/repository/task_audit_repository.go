package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"rainchanel.com/internal/database"
)

type TaskAuditRepository interface {
	CreateTaskAudit(audit *database.TaskAudit) error
	FindTaskAuditByTaskID(taskID uint) (*database.TaskAudit, error)
	UpdateTaskAuditStatus(taskID uint, status database.TaskStatus) error
	UpdateTaskAuditConsumed(taskID uint) error
	UpdateTaskAuditCompleted(taskID uint, processedBy uint) error
	FindAndClaimPendingTask() (*database.TaskAudit, error)
}

type taskAuditRepository struct{}

func NewTaskAuditRepository() TaskAuditRepository {
	return &taskAuditRepository{}
}

func (r *taskAuditRepository) CreateTaskAudit(audit *database.TaskAudit) error {
	return database.DB.Create(audit).Error
}

func (r *taskAuditRepository) FindTaskAuditByTaskID(taskID uint) (*database.TaskAudit, error) {
	var audit database.TaskAudit
	err := database.DB.Preload("Task").Where("task_id = ?", taskID).First(&audit).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &audit, nil
}

func (r *taskAuditRepository) UpdateTaskAuditStatus(taskID uint, status database.TaskStatus) error {
	return database.DB.Model(&database.TaskAudit{}).
		Where("task_id = ?", taskID).
		Update("status", status).Error
}

func (r *taskAuditRepository) UpdateTaskAuditConsumed(taskID uint) error {
	now := time.Now()
	return database.DB.Model(&database.TaskAudit{}).
		Where("task_id = ? AND consumed_at IS NULL", taskID).
		Updates(map[string]interface{}{
			"status":      database.TaskStatusProcessing,
			"consumed_at": now,
		}).Error
}

func (r *taskAuditRepository) UpdateTaskAuditCompleted(taskID uint, processedBy uint) error {
	now := time.Now()
	return database.DB.Model(&database.TaskAudit{}).
		Where("task_id = ?", taskID).
		Updates(map[string]interface{}{
			"status":       database.TaskStatusCompleted,
			"completed_at": now,
			"processed_by": processedBy,
		}).Error
}

func (r *taskAuditRepository) FindAndClaimPendingTask() (*database.TaskAudit, error) {
	var audit database.TaskAudit
	
	// Use a transaction with SELECT FOR UPDATE to prevent race conditions
	// This ensures only one worker can claim a task at a time
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	// Find oldest pending task and lock it
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("status = ?", database.TaskStatusPending).
		Order("published_at ASC").
		Preload("Task").
		First(&audit).Error
	
	if err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	
	// Claim the task by updating status and consumed_at
	now := time.Now()
	err = tx.Model(&audit).
		Updates(map[string]interface{}{
			"status":      database.TaskStatusProcessing,
			"consumed_at": now,
		}).Error
	
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	
	// Reload to get the updated audit
	err = database.DB.Preload("Task").Where("task_id = ?", audit.TaskID).First(&audit).Error
	if err != nil {
		return nil, err
	}
	
	return &audit, nil
}


