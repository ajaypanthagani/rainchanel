package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
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
	ConsumeResult(userID uint) (*dto.Result, error)
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

func (s *taskService) PublishTask(task dto.Task, createdBy uint) (uint, error) {
	// Validate the WASM task before creating it
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

	// Task is now in database with status 'pending', ready for workers to consume
	return taskID, nil
}

func (s *taskService) ConsumeTask() (*dto.Task, error) {
	// Find and claim a pending task from database
	audit, err := s.auditRepo.FindAndClaimPendingTask()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoTasksAvailable
		}
		return nil, fmt.Errorf("failed to find and claim task: %w", err)
	}

	// Parse task args from JSON string
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

	// Store result in database
	dbResult := &database.Result{
		TaskID:      taskID,
		CreatedBy:   createdBy,
		ProcessedBy: processedBy,
		Result:      result,
	}

	if err := s.resultRepo.CreateResult(dbResult); err != nil {
		_ = s.auditRepo.UpdateTaskAuditStatus(taskID, database.TaskStatusFailed)
		return fmt.Errorf("failed to create result in database: %w", err)
	}

	return nil
}

func (s *taskService) ConsumeResult(userID uint) (*dto.Result, error) {
	// Query from database
	dbResult, err := s.resultRepo.FindOldestUnconsumedResultByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoTasksAvailable
		}
		return nil, fmt.Errorf("failed to find result: %w", err)
	}

	// Parse the result JSON string into the DTO format
	var resultData interface{}
	if err := json.Unmarshal([]byte(dbResult.Result), &resultData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result data: %w", err)
	}

	result := &dto.Result{
		TaskID:    dbResult.TaskID,
		CreatedBy: dbResult.CreatedBy,
		Result:    resultData,
	}

	// Mark as consumed (set consumed flag to true)
	if err := s.resultRepo.MarkResultAsConsumed(dbResult.ID); err != nil {
		// Log error but don't fail - result was already returned
		log.Printf("Warning: failed to mark result as consumed: %v", err)
	}

	return result, nil
}
