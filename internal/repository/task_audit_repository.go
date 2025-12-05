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
	GetEnhancedStatistics() (map[string]interface{}, error)
	FindTasksWithPagination(limit, offset int, status *database.TaskStatus) ([]*database.TaskAudit, int64, error)
	GetRecentActivity(hours int) (map[string]int64, error)
	GetErrorBreakdown(limit int) ([]map[string]interface{}, error)
	GetUserStatistics(userID uint) (map[string]int64, error)
	GetUserEnhancedStatistics(userID uint) (map[string]interface{}, error)
	FindUserTasksWithPagination(userID uint, limit, offset int, status *database.TaskStatus) ([]*database.TaskAudit, int64, error)
	GetUserRecentActivity(userID uint, hours int) (map[string]int64, error)
	GetUserErrorBreakdown(userID uint, limit int) ([]map[string]interface{}, error)
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

func (r *taskAuditRepository) GetEnhancedStatistics() (map[string]interface{}, error) {
	if database.DB == nil {
		return nil, errors.New("database not initialized")
	}

	stats := make(map[string]interface{})

	basicStats, err := r.GetTaskStatistics()
	if err != nil {
		return nil, err
	}
	stats["counts"] = basicStats

	var totalTasks int64
	if err := database.DB.Model(&database.TaskAudit{}).Count(&totalTasks).Error; err != nil {
		return nil, err
	}
	stats["total"] = totalTasks

	var completedLastHour int64
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	if err := database.DB.Model(&database.TaskAudit{}).
		Where("status = ? AND completed_at > ?", database.TaskStatusCompleted, oneHourAgo).
		Count(&completedLastHour).Error; err != nil {
		return nil, err
	}
	stats["completed_last_hour"] = completedLastHour

	var avgProcessingTime float64
	if err := database.DB.Model(&database.TaskAudit{}).
		Where("status = ? AND consumed_at IS NOT NULL AND completed_at IS NOT NULL", database.TaskStatusCompleted).
		Select("AVG(TIMESTAMPDIFF(SECOND, consumed_at, completed_at))").
		Scan(&avgProcessingTime).Error; err != nil {

		avgProcessingTime = 0
	}
	stats["avg_processing_time_seconds"] = avgProcessingTime

	var retriedTasks int64
	if err := database.DB.Model(&database.TaskAudit{}).
		Where("retry_count > 0").
		Count(&retriedTasks).Error; err != nil {
		return nil, err
	}
	stats["retried_tasks"] = retriedTasks

	var totalRetries int64
	if err := database.DB.Model(&database.TaskAudit{}).
		Select("SUM(retry_count)").
		Scan(&totalRetries).Error; err != nil {
		totalRetries = 0
	}
	stats["total_retries"] = totalRetries

	return stats, nil
}

func (r *taskAuditRepository) FindTasksWithPagination(limit, offset int, status *database.TaskStatus) ([]*database.TaskAudit, int64, error) {
	if database.DB == nil {
		return nil, 0, errors.New("database not initialized")
	}

	var audits []*database.TaskAudit
	var total int64

	query := database.DB.Model(&database.TaskAudit{}).Preload("Task").Preload("Task.Creator").Preload("Worker")

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("published_at DESC").Limit(limit).Offset(offset).Find(&audits).Error; err != nil {
		return nil, 0, err
	}

	return audits, total, nil
}

func (r *taskAuditRepository) GetRecentActivity(hours int) (map[string]int64, error) {
	if database.DB == nil {
		return nil, errors.New("database not initialized")
	}

	activity := make(map[string]int64)
	threshold := time.Now().Add(-time.Duration(hours) * time.Hour)

	var published int64
	if err := database.DB.Model(&database.TaskAudit{}).
		Where("published_at > ?", threshold).
		Count(&published).Error; err != nil {
		return nil, err
	}
	activity["published"] = published

	var completed int64
	if err := database.DB.Model(&database.TaskAudit{}).
		Where("status = ? AND completed_at > ?", database.TaskStatusCompleted, threshold).
		Count(&completed).Error; err != nil {
		return nil, err
	}
	activity["completed"] = completed

	var failed int64
	if err := database.DB.Model(&database.TaskAudit{}).
		Where("status = ? AND updated_at > ?", database.TaskStatusFailed, threshold).
		Count(&failed).Error; err != nil {
		return nil, err
	}
	activity["failed"] = failed

	return activity, nil
}

func (r *taskAuditRepository) GetErrorBreakdown(limit int) ([]map[string]interface{}, error) {
	if database.DB == nil {
		return nil, errors.New("database not initialized")
	}

	var results []struct {
		ErrorMsg string `gorm:"column:error_msg"`
		Count    int64  `gorm:"column:count"`
	}

	if err := database.DB.Model(&database.TaskAudit{}).
		Select("error_msg, COUNT(*) as count").
		Where("status = ? AND error_msg != ''", database.TaskStatusFailed).
		Group("error_msg").
		Order("count DESC").
		Limit(limit).
		Scan(&results).Error; err != nil {
		return nil, err
	}

	breakdown := make([]map[string]interface{}, len(results))
	for i, result := range results {
		breakdown[i] = map[string]interface{}{
			"error": result.ErrorMsg,
			"count": result.Count,
		}
	}

	return breakdown, nil
}

func (r *taskAuditRepository) GetUserStatistics(userID uint) (map[string]int64, error) {
	if database.DB == nil {
		return nil, errors.New("database not initialized")
	}
	stats := make(map[string]int64)

	var count int64

	if err := database.DB.Model(&database.TaskAudit{}).
		Joins("JOIN tasks ON task_audit.task_id = tasks.id").
		Where("tasks.created_by = ? AND task_audit.status = ?", userID, database.TaskStatusPending).
		Count(&count).Error; err != nil {
		return nil, err
	}
	stats["pending"] = count

	if err := database.DB.Model(&database.TaskAudit{}).
		Joins("JOIN tasks ON task_audit.task_id = tasks.id").
		Where("tasks.created_by = ? AND task_audit.status = ?", userID, database.TaskStatusProcessing).
		Count(&count).Error; err != nil {
		return nil, err
	}
	stats["processing"] = count

	if err := database.DB.Model(&database.TaskAudit{}).
		Joins("JOIN tasks ON task_audit.task_id = tasks.id").
		Where("tasks.created_by = ? AND task_audit.status = ?", userID, database.TaskStatusCompleted).
		Count(&count).Error; err != nil {
		return nil, err
	}
	stats["completed"] = count

	if err := database.DB.Model(&database.TaskAudit{}).
		Joins("JOIN tasks ON task_audit.task_id = tasks.id").
		Where("tasks.created_by = ? AND task_audit.status = ?", userID, database.TaskStatusFailed).
		Count(&count).Error; err != nil {
		return nil, err
	}
	stats["failed"] = count

	return stats, nil
}

func (r *taskAuditRepository) GetUserEnhancedStatistics(userID uint) (map[string]interface{}, error) {
	if database.DB == nil {
		return nil, errors.New("database not initialized")
	}

	stats := make(map[string]interface{})

	basicStats, err := r.GetUserStatistics(userID)
	if err != nil {
		return nil, err
	}
	stats["counts"] = basicStats

	var totalTasks int64
	if err := database.DB.Model(&database.TaskAudit{}).
		Joins("JOIN tasks ON task_audit.task_id = tasks.id").
		Where("tasks.created_by = ?", userID).
		Count(&totalTasks).Error; err != nil {
		return nil, err
	}
	stats["total"] = totalTasks

	var completedLastHour int64
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	if err := database.DB.Model(&database.TaskAudit{}).
		Joins("JOIN tasks ON task_audit.task_id = tasks.id").
		Where("tasks.created_by = ? AND task_audit.status = ? AND task_audit.completed_at > ?", userID, database.TaskStatusCompleted, oneHourAgo).
		Count(&completedLastHour).Error; err != nil {
		return nil, err
	}
	stats["completed_last_hour"] = completedLastHour

	var avgProcessingTime float64
	if err := database.DB.Model(&database.TaskAudit{}).
		Joins("JOIN tasks ON task_audit.task_id = tasks.id").
		Where("tasks.created_by = ? AND task_audit.status = ? AND task_audit.consumed_at IS NOT NULL AND task_audit.completed_at IS NOT NULL", userID, database.TaskStatusCompleted).
		Select("AVG(TIMESTAMPDIFF(SECOND, task_audit.consumed_at, task_audit.completed_at))").
		Scan(&avgProcessingTime).Error; err != nil {
		avgProcessingTime = 0
	}
	stats["avg_processing_time_seconds"] = avgProcessingTime

	var retriedTasks int64
	if err := database.DB.Model(&database.TaskAudit{}).
		Joins("JOIN tasks ON task_audit.task_id = tasks.id").
		Where("tasks.created_by = ? AND task_audit.retry_count > 0", userID).
		Count(&retriedTasks).Error; err != nil {
		return nil, err
	}
	stats["retried_tasks"] = retriedTasks

	var totalRetries int64
	if err := database.DB.Model(&database.TaskAudit{}).
		Joins("JOIN tasks ON task_audit.task_id = tasks.id").
		Where("tasks.created_by = ?", userID).
		Select("SUM(task_audit.retry_count)").
		Scan(&totalRetries).Error; err != nil {
		totalRetries = 0
	}
	stats["total_retries"] = totalRetries

	return stats, nil
}

func (r *taskAuditRepository) FindUserTasksWithPagination(userID uint, limit, offset int, status *database.TaskStatus) ([]*database.TaskAudit, int64, error) {
	if database.DB == nil {
		return nil, 0, errors.New("database not initialized")
	}

	var audits []*database.TaskAudit
	var total int64

	query := database.DB.Model(&database.TaskAudit{}).
		Joins("JOIN tasks ON task_audit.task_id = tasks.id").
		Where("tasks.created_by = ?", userID).
		Preload("Task").Preload("Task.Creator").Preload("Worker")

	if status != nil {
		query = query.Where("task_audit.status = ?", *status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("task_audit.published_at DESC").Limit(limit).Offset(offset).Find(&audits).Error; err != nil {
		return nil, 0, err
	}

	return audits, total, nil
}

func (r *taskAuditRepository) GetUserRecentActivity(userID uint, hours int) (map[string]int64, error) {
	if database.DB == nil {
		return nil, errors.New("database not initialized")
	}

	activity := make(map[string]int64)
	threshold := time.Now().Add(-time.Duration(hours) * time.Hour)

	var published int64
	if err := database.DB.Model(&database.TaskAudit{}).
		Joins("JOIN tasks ON task_audit.task_id = tasks.id").
		Where("tasks.created_by = ? AND task_audit.published_at > ?", userID, threshold).
		Count(&published).Error; err != nil {
		return nil, err
	}
	activity["published"] = published

	var completed int64
	if err := database.DB.Model(&database.TaskAudit{}).
		Joins("JOIN tasks ON task_audit.task_id = tasks.id").
		Where("tasks.created_by = ? AND task_audit.status = ? AND task_audit.completed_at > ?", userID, database.TaskStatusCompleted, threshold).
		Count(&completed).Error; err != nil {
		return nil, err
	}
	activity["completed"] = completed

	var failed int64
	if err := database.DB.Model(&database.TaskAudit{}).
		Joins("JOIN tasks ON task_audit.task_id = tasks.id").
		Where("tasks.created_by = ? AND task_audit.status = ? AND task_audit.updated_at > ?", userID, database.TaskStatusFailed, threshold).
		Count(&failed).Error; err != nil {
		return nil, err
	}
	activity["failed"] = failed

	return activity, nil
}

func (r *taskAuditRepository) GetUserErrorBreakdown(userID uint, limit int) ([]map[string]interface{}, error) {
	if database.DB == nil {
		return nil, errors.New("database not initialized")
	}

	var results []struct {
		ErrorMsg string `gorm:"column:error_msg"`
		Count    int64  `gorm:"column:count"`
	}

	if err := database.DB.Model(&database.TaskAudit{}).
		Joins("JOIN tasks ON task_audit.task_id = tasks.id").
		Select("task_audit.error_msg, COUNT(*) as count").
		Where("tasks.created_by = ? AND task_audit.status = ? AND task_audit.error_msg != ''", userID, database.TaskStatusFailed).
		Group("task_audit.error_msg").
		Order("count DESC").
		Limit(limit).
		Scan(&results).Error; err != nil {
		return nil, err
	}

	breakdown := make([]map[string]interface{}, len(results))
	for i, result := range results {
		breakdown[i] = map[string]interface{}{
			"error": result.ErrorMsg,
			"count": result.Count,
		}
	}

	return breakdown, nil
}
