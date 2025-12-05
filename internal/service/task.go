package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"rainchanel.com/internal/config"
	"rainchanel.com/internal/database"
	"rainchanel.com/internal/dto"
	"rainchanel.com/internal/repository"
	"rainchanel.com/internal/validation"
)

var ErrNoTasksAvailable = errors.New("no tasks available")
var ErrTaskNotFound = errors.New("task not found")
var ErrInvalidCreatedBy = errors.New("created_by does not match task record")

type TaskService interface {
	PublishTask(task dto.Task, createdBy uint) (uint, error)
	ConsumeTask() (*dto.Task, error)
	PublishResult(taskID uint, createdBy uint, processedBy uint, result string) error
	PublishFailure(taskID uint, createdBy uint, processedBy uint, errorMsg string) error
	ConsumeResult(userID uint) (*dto.Result, error)
	ReclaimStaleTasks() (int, error)
}

type taskService struct {
	taskRepo   repository.TaskRepository
	auditRepo  repository.TaskAuditRepository
	resultRepo repository.ResultRepository
}

func NewTaskService() TaskService {
	return &taskService{
		taskRepo:   repository.NewTaskRepository(),
		auditRepo:  repository.NewTaskAuditRepository(),
		resultRepo: repository.NewResultRepository(),
	}
}

func NewTaskServiceWithRepos(taskRepo repository.TaskRepository, auditRepo repository.TaskAuditRepository, resultRepo repository.ResultRepository) TaskService {
	return &taskService{
		taskRepo:   taskRepo,
		auditRepo:  auditRepo,
		resultRepo: resultRepo,
	}
}

func (s *taskService) PublishTask(task dto.Task, createdBy uint) (uint, error) {

	if err := validation.ValidateTask(task.WasmModule, task.Func, task.Args); err != nil {
		return 0, fmt.Errorf("task validation failed: %w", err)
	}

	argsJSON, err := json.Marshal(task.Args)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal task args: %w", err)
	}

	dbTask := &database.Task{
		WasmModule: task.WasmModule,
		Func:       task.Func,
		Args:       string(argsJSON),
		CreatedBy:  createdBy,
	}
	if err := s.taskRepo.CreateTask(dbTask); err != nil {
		return 0, fmt.Errorf("failed to create task in database: %w", err)
	}

	taskID := dbTask.ID
	task.ID = taskID
	task.CreatedBy = createdBy

	audit := &database.TaskAudit{
		TaskID:      taskID,
		Status:      database.TaskStatusPending,
		PublishedAt: time.Now(),
	}

	if err := s.auditRepo.CreateTaskAudit(audit); err != nil {
		return 0, fmt.Errorf("failed to create task audit: %w", err)
	}

	return taskID, nil
}

func (s *taskService) ConsumeTask() (*dto.Task, error) {

	audit, err := s.auditRepo.FindAndClaimPendingTask()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoTasksAvailable
		}
		return nil, fmt.Errorf("failed to find and claim task: %w", err)
	}

	var args interface{}
	if audit.Task.Args != "" {
		if err := json.Unmarshal([]byte(audit.Task.Args), &args); err != nil {
			return nil, fmt.Errorf("failed to unmarshal task args: %w", err)
		}
	}

	task := &dto.Task{
		ID:         audit.Task.ID,
		WasmModule: audit.Task.WasmModule,
		Func:       audit.Task.Func,
		Args:       args,
		CreatedBy:  audit.Task.CreatedBy,
	}

	return task, nil
}

func (s *taskService) PublishResult(taskID uint, createdBy uint, processedBy uint, result string) error {
	audit, err := s.auditRepo.FindTaskAuditByTaskID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTaskNotFound
		}
		return fmt.Errorf("failed to find task audit: %w", err)
	}

	if audit.Task.CreatedBy != createdBy {
		return ErrInvalidCreatedBy
	}

	if err := s.auditRepo.UpdateTaskAuditCompleted(taskID, processedBy); err != nil {
		return fmt.Errorf("failed to update task audit: %w", err)
	}

	dbResult := &database.Result{
		TaskID:      taskID,
		CreatedBy:   createdBy,
		ProcessedBy: processedBy,
		Result:      result,
	}

	if err := s.resultRepo.CreateResult(dbResult); err != nil {

		if rollbackErr := s.auditRepo.UpdateTaskAuditStatus(taskID, database.TaskStatusFailed); rollbackErr != nil {
			logrus.WithFields(logrus.Fields{
				"task_id":      taskID,
				"rollback_err": rollbackErr.Error(),
				"original_err": err.Error(),
			}).Warn("Failed to rollback task status after result creation failure")
		}
		return fmt.Errorf("failed to create result in database: %w", err)
	}

	return nil
}

func (s *taskService) PublishFailure(taskID uint, createdBy uint, processedBy uint, errorMsg string) error {
	audit, err := s.auditRepo.FindTaskAuditByTaskID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTaskNotFound
		}
		return fmt.Errorf("failed to find task audit: %w", err)
	}

	if audit.Task.CreatedBy != createdBy {
		return ErrInvalidCreatedBy
	}

	maxRetries := config.App.Task.MaxRetries
	if audit.RetryCount < maxRetries {

		backoffSeconds := int(math.Pow(2, float64(audit.RetryCount)))
		errorMsgWithRetry := fmt.Sprintf("Task failed (attempt %d/%d): %s. Will retry after backoff.",
			audit.RetryCount+1, maxRetries+1, errorMsg)

		if err := s.auditRepo.ReclaimStaleTask(taskID, errorMsgWithRetry); err != nil {
			return fmt.Errorf("failed to reclaim task for retry: %w", err)
		}

		logrus.WithFields(logrus.Fields{
			"task_id":         taskID,
			"attempt":         audit.RetryCount + 1,
			"max_retries":     maxRetries + 1,
			"backoff_seconds": backoffSeconds,
			"error":           errorMsg,
		}).Info("Task failed, retrying")
		return nil
	}

	if err := s.auditRepo.UpdateTaskFailed(taskID, fmt.Sprintf("Task failed after %d retries: %s", maxRetries+1, errorMsg)); err != nil {
		return fmt.Errorf("failed to update task as failed: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"task_id":     taskID,
		"retry_count": maxRetries + 1,
		"error":       errorMsg,
	}).Error("Task failed permanently")
	return nil
}

func (s *taskService) ReclaimStaleTasks() (int, error) {
	timeoutDuration := time.Duration(config.App.Task.TimeoutSeconds) * time.Second
	staleTasks, err := s.auditRepo.FindStaleTasks(timeoutDuration)
	if err != nil {
		return 0, fmt.Errorf("failed to find stale tasks: %w", err)
	}

	reclaimedCount := 0
	maxRetries := config.App.Task.MaxRetries

	for _, audit := range staleTasks {
		if audit.RetryCount >= maxRetries {

			errorMsg := fmt.Sprintf("Task timed out after %d retries (exceeded %d seconds)",
				audit.RetryCount, config.App.Task.TimeoutSeconds)
			if err := s.auditRepo.UpdateTaskFailed(audit.TaskID, errorMsg); err != nil {
				logrus.WithFields(logrus.Fields{
					"task_id": audit.TaskID,
					"error":   err.Error(),
				}).Error("Failed to mark stale task as failed")
				continue
			}
			logrus.WithFields(logrus.Fields{
				"task_id":     audit.TaskID,
				"retry_count": audit.RetryCount,
			}).Warn("Marked stale task as failed (max retries exceeded)")
		} else {

			errorMsg := fmt.Sprintf("Task timed out (exceeded %d seconds), reclaiming for retry",
				config.App.Task.TimeoutSeconds)
			if err := s.auditRepo.ReclaimStaleTask(audit.TaskID, errorMsg); err != nil {
				logrus.WithFields(logrus.Fields{
					"task_id": audit.TaskID,
					"error":   err.Error(),
				}).Error("Failed to reclaim stale task")
				continue
			}
			reclaimedCount++
			logrus.WithFields(logrus.Fields{
				"task_id":     audit.TaskID,
				"attempt":     audit.RetryCount + 1,
				"max_retries": maxRetries + 1,
			}).Info("Reclaimed stale task for retry")
		}
	}

	return reclaimedCount, nil
}

func (s *taskService) ConsumeResult(userID uint) (*dto.Result, error) {

	dbResult, err := s.resultRepo.FindOldestUnconsumedResultByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoTasksAvailable
		}
		return nil, fmt.Errorf("failed to find result: %w", err)
	}

	var resultData interface{}
	if err := json.Unmarshal([]byte(dbResult.Result), &resultData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result data: %w", err)
	}

	result := &dto.Result{
		TaskID:    dbResult.TaskID,
		CreatedBy: dbResult.CreatedBy,
		Result:    resultData,
	}

	if err := s.resultRepo.MarkResultAsConsumed(dbResult.ID); err != nil {
		// Log error but don't fail - result was already returned
		logrus.WithFields(logrus.Fields{
			"result_id": dbResult.ID,
			"error":     err.Error(),
		}).Warn("Failed to mark result as consumed")
	}

	return result, nil
}
