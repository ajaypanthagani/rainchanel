package repository

import (
	"errors"
	"fmt"
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
	FindStaleTasks(timeoutDuration time.Duration) ([]*database.TaskAudit, error)
	ReclaimStaleTask(taskID uint, errorMsg string) error
	UpdateTaskFailed(taskID uint, errorMsg string) error
	GetTaskStatistics() (map[string]int64, error)
}

type taskAuditRepository struct{}

func NewTaskAuditRepository() TaskAuditRepository {
	return &taskAuditRepository{}
}

func (r *taskAuditRepository) CreateTaskAudit(audit *database.TaskAudit) error {
	if database.DB == nil {
		return errors.New("database not initialized")
	}
	return database.DB.Create(audit).Error
}

func (r *taskAuditRepository) FindTaskAuditByTaskID(taskID uint) (*database.TaskAudit, error) {
	if database.DB == nil {
		return nil, errors.New("database not initialized")
	}
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
	if database.DB == nil {
		return errors.New("database not initialized")
	}
	return database.DB.Model(&database.TaskAudit{}).
		Where("task_id = ?", taskID).
		Update("status", status).Error
}

func (r *taskAuditRepository) UpdateTaskAuditConsumed(taskID uint) error {
	if database.DB == nil {
		return errors.New("database not initialized")
	}
	now := time.Now()
	return database.DB.Model(&database.TaskAudit{}).
		Where("task_id = ? AND consumed_at IS NULL", taskID).
		Updates(map[string]interface{}{
			"status":      database.TaskStatusProcessing,
			"consumed_at": now,
		}).Error
}

func (r *taskAuditRepository) UpdateTaskAuditCompleted(taskID uint, processedBy uint) error {
	if database.DB == nil {
		return errors.New("database not initialized")
	}
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
	if database.DB == nil {
		return nil, errors.New("database not initialized")
	}

	sqlDB, err := database.DB.DB()
	if err != nil {
		return nil, fmt.Errorf("database not initialized: %w", err)
	}
	if sqlDB == nil {
		return nil, errors.New("database not initialized")
	}

	var audit database.TaskAudit

	tx := database.DB.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	committed := false
	defer func() {
		if !committed {
			if r := recover(); r != nil {
				tx.Rollback()
				panic(r)
			} else if tx.Error == nil {
				tx.Rollback()
			}
		}
	}()

	err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("status = ?", database.TaskStatusPending).
		Order("published_at ASC").
		Preload("Task").
		First(&audit).Error

	if err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to find pending task: %w", err)
	}

	now := time.Now()
	err = tx.Model(&audit).
		Updates(map[string]interface{}{
			"status":      database.TaskStatusProcessing,
			"consumed_at": now,
		}).Error

	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to claim task: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	committed = true

	err = database.DB.Preload("Task").Where("task_id = ?", audit.TaskID).First(&audit).Error
	if err != nil {
		return nil, fmt.Errorf("failed to reload task audit: %w", err)
	}

	return &audit, nil
}

func (r *taskAuditRepository) FindStaleTasks(timeoutDuration time.Duration) ([]*database.TaskAudit, error) {
	if database.DB == nil {
		return nil, errors.New("database not initialized")
	}
	var audits []*database.TaskAudit
	threshold := time.Now().Add(-timeoutDuration)

	err := database.DB.
		Where("status = ? AND consumed_at < ?", database.TaskStatusProcessing, threshold).
		Preload("Task").
		Find(&audits).Error

	if err != nil {
		return nil, err
	}

	return audits, nil
}

func (r *taskAuditRepository) ReclaimStaleTask(taskID uint, errorMsg string) error {
	if database.DB == nil {
		return errors.New("database not initialized")
	}
	return database.DB.Model(&database.TaskAudit{}).
		Where("task_id = ?", taskID).
		Updates(map[string]interface{}{
			"status":      database.TaskStatusPending,
			"consumed_at": nil,
			"error_msg":   errorMsg,
			"retry_count": gorm.Expr("retry_count + 1"),
		}).Error
}

func (r *taskAuditRepository) UpdateTaskFailed(taskID uint, errorMsg string) error {
	if database.DB == nil {
		return errors.New("database not initialized")
	}
	return database.DB.Model(&database.TaskAudit{}).
		Where("task_id = ?", taskID).
		Updates(map[string]interface{}{
			"status":    database.TaskStatusFailed,
			"error_msg": errorMsg,
		}).Error
}

func (r *taskAuditRepository) GetTaskStatistics() (map[string]int64, error) {
	if database.DB == nil {
		return nil, errors.New("database not initialized")
	}
	stats := make(map[string]int64)

	var count int64

	if err := database.DB.Model(&database.TaskAudit{}).
		Where("status = ?", database.TaskStatusPending).
		Count(&count).Error; err != nil {
		return nil, err
	}
	stats["pending"] = count

	if err := database.DB.Model(&database.TaskAudit{}).
		Where("status = ?", database.TaskStatusProcessing).
		Count(&count).Error; err != nil {
		return nil, err
	}
	stats["processing"] = count

	if err := database.DB.Model(&database.TaskAudit{}).
		Where("status = ?", database.TaskStatusCompleted).
		Count(&count).Error; err != nil {
		return nil, err
	}
	stats["completed"] = count

	if err := database.DB.Model(&database.TaskAudit{}).
		Where("status = ?", database.TaskStatusFailed).
		Count(&count).Error; err != nil {
		return nil, err
	}
	stats["failed"] = count

	return stats, nil
}
