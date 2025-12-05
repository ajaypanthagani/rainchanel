package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"rainchanel.com/internal/config"
	"rainchanel.com/internal/database"
	"rainchanel.com/internal/dto"
)

func TestNewTaskService(t *testing.T) {
	service := NewTaskService()

	if service == nil {
		t.Error("NewTaskService() returned nil")
	}
}

func TestTaskService_PublishTask(t *testing.T) {
	tests := []struct {
		name       string
		task       dto.Task
		createdBy  uint
		wantErr    bool
		wantTaskID uint
		setupMocks func() (*MockTaskRepository, *MockTaskAuditRepository, *MockResultRepository)
	}{
		{
			name: "validation error - invalid WASM module",
			task: dto.Task{
				ID:         0,
				WasmModule: "AGFzbQEAAAABBwFgAn9/AX9gAAF/",
				Func:       "testFunc",
				Args:       []string{"arg1"},
			},
			createdBy:  1,
			wantErr:    true,
			wantTaskID: 0,
			setupMocks: func() (*MockTaskRepository, *MockTaskAuditRepository, *MockResultRepository) {
				return &MockTaskRepository{}, &MockTaskAuditRepository{}, &MockResultRepository{}
			},
		},
		{
			name: "validation error",
			task: dto.Task{
				ID:         0,
				WasmModule: "invalid",
				Func:       "testFunc",
				Args:       []string{"arg1"},
			},
			createdBy:  1,
			wantErr:    true,
			wantTaskID: 0,
			setupMocks: func() (*MockTaskRepository, *MockTaskAuditRepository, *MockResultRepository) {
				return &MockTaskRepository{}, &MockTaskAuditRepository{}, &MockResultRepository{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskRepo, auditRepo, resultRepo := tt.setupMocks()
			service := NewTaskServiceWithRepos(taskRepo, auditRepo, resultRepo)

			taskID, err := service.PublishTask(tt.task, tt.createdBy)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.wantTaskID != 0 {
					assert.Equal(t, tt.wantTaskID, taskID)
				} else {
					assert.NotZero(t, taskID)
				}
			}
		})
	}
}

func TestTaskService_ConsumeTask(t *testing.T) {
	tests := []struct {
		name       string
		wantErr    bool
		setupMocks func() (*MockTaskRepository, *MockTaskAuditRepository, *MockResultRepository)
	}{
		{
			name:    "no tasks available",
			wantErr: true,
			setupMocks: func() (*MockTaskRepository, *MockTaskAuditRepository, *MockResultRepository) {
				auditRepo := &MockTaskAuditRepository{
					FindAndClaimPendingTaskFunc: func() (*database.TaskAudit, error) {
						return nil, gorm.ErrRecordNotFound
					},
				}
				return &MockTaskRepository{}, auditRepo, &MockResultRepository{}
			},
		},
		{
			name:    "success",
			wantErr: false,
			setupMocks: func() (*MockTaskRepository, *MockTaskAuditRepository, *MockResultRepository) {
				auditRepo := &MockTaskAuditRepository{
					FindAndClaimPendingTaskFunc: func() (*database.TaskAudit, error) {
						return &database.TaskAudit{
							TaskID: 1,
							Task: database.Task{
								ID:         1,
								WasmModule: "AGFzbQEAAAABBwFgAn9/AX9gAAF/",
								Func:       "testFunc",
								Args:       `["arg1"]`,
								CreatedBy:  1,
							},
						}, nil
					},
				}
				return &MockTaskRepository{}, auditRepo, &MockResultRepository{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskRepo, auditRepo, resultRepo := tt.setupMocks()
			service := NewTaskServiceWithRepos(taskRepo, auditRepo, resultRepo)

			task, err := service.ConsumeTask()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, task)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, task)
			}
		})
	}
}

func TestTaskService_PublishResult(t *testing.T) {
	tests := []struct {
		name        string
		taskID      uint
		createdBy   uint
		processedBy uint
		result      string
		wantErr     bool
		setupMocks  func() (*MockTaskRepository, *MockTaskAuditRepository, *MockResultRepository)
	}{
		{
			name:        "task not found",
			taskID:      123,
			createdBy:   1,
			processedBy: 2,
			result:      `{"result":"success"}`,
			wantErr:     true,
			setupMocks: func() (*MockTaskRepository, *MockTaskAuditRepository, *MockResultRepository) {
				auditRepo := &MockTaskAuditRepository{
					FindTaskAuditByTaskIDFunc: func(taskID uint) (*database.TaskAudit, error) {
						return nil, gorm.ErrRecordNotFound
					},
				}
				return &MockTaskRepository{}, auditRepo, &MockResultRepository{}
			},
		},
		{
			name:        "invalid created_by",
			taskID:      123,
			createdBy:   2,
			processedBy: 2,
			result:      `{"result":"success"}`,
			wantErr:     true,
			setupMocks: func() (*MockTaskRepository, *MockTaskAuditRepository, *MockResultRepository) {
				auditRepo := &MockTaskAuditRepository{
					FindTaskAuditByTaskIDFunc: func(taskID uint) (*database.TaskAudit, error) {
						return &database.TaskAudit{
							TaskID: 123,
							Task: database.Task{
								ID:        123,
								CreatedBy: 1,
							},
						}, nil
					},
				}
				return &MockTaskRepository{}, auditRepo, &MockResultRepository{}
			},
		},
		{
			name:        "success",
			taskID:      123,
			createdBy:   1,
			processedBy: 2,
			result:      `{"result":"success"}`,
			wantErr:     false,
			setupMocks: func() (*MockTaskRepository, *MockTaskAuditRepository, *MockResultRepository) {
				auditRepo := &MockTaskAuditRepository{
					FindTaskAuditByTaskIDFunc: func(taskID uint) (*database.TaskAudit, error) {
						return &database.TaskAudit{
							TaskID: 123,
							Task: database.Task{
								ID:        123,
								CreatedBy: 1,
							},
						}, nil
					},
					UpdateTaskAuditCompletedFunc: func(taskID uint, processedBy uint) error {
						return nil
					},
				}
				resultRepo := &MockResultRepository{
					CreateResultFunc: func(result *database.Result) error {
						return nil
					},
				}
				return &MockTaskRepository{}, auditRepo, resultRepo
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskRepo, auditRepo, resultRepo := tt.setupMocks()
			service := NewTaskServiceWithRepos(taskRepo, auditRepo, resultRepo)

			err := service.PublishResult(tt.taskID, tt.createdBy, tt.processedBy, tt.result)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTaskService_PublishFailure(t *testing.T) {

	config.App = &config.Config{
		Task: config.TaskConfig{
			MaxRetries: 3,
		},
	}

	tests := []struct {
		name        string
		taskID      uint
		createdBy   uint
		processedBy uint
		errorMsg    string
		retryCount  int
		wantErr     bool
		setupMocks  func() (*MockTaskRepository, *MockTaskAuditRepository, *MockResultRepository)
	}{
		{
			name:        "retry available",
			taskID:      123,
			createdBy:   1,
			processedBy: 2,
			errorMsg:    "execution failed",
			retryCount:  1,
			wantErr:     false,
			setupMocks: func() (*MockTaskRepository, *MockTaskAuditRepository, *MockResultRepository) {
				auditRepo := &MockTaskAuditRepository{
					FindTaskAuditByTaskIDFunc: func(taskID uint) (*database.TaskAudit, error) {
						return &database.TaskAudit{
							TaskID:     123,
							RetryCount: 1,
							Task: database.Task{
								ID:        123,
								CreatedBy: 1,
							},
						}, nil
					},
					ReclaimStaleTaskFunc: func(taskID uint, errorMsg string) error {
						return nil
					},
				}
				return &MockTaskRepository{}, auditRepo, &MockResultRepository{}
			},
		},
		{
			name:        "max retries exceeded",
			taskID:      123,
			createdBy:   1,
			processedBy: 2,
			errorMsg:    "execution failed",
			retryCount:  3,
			wantErr:     false,
			setupMocks: func() (*MockTaskRepository, *MockTaskAuditRepository, *MockResultRepository) {
				auditRepo := &MockTaskAuditRepository{
					FindTaskAuditByTaskIDFunc: func(taskID uint) (*database.TaskAudit, error) {
						return &database.TaskAudit{
							TaskID:     123,
							RetryCount: 3,
							Task: database.Task{
								ID:        123,
								CreatedBy: 1,
							},
						}, nil
					},
					UpdateTaskFailedFunc: func(taskID uint, errorMsg string) error {
						return nil
					},
				}
				return &MockTaskRepository{}, auditRepo, &MockResultRepository{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskRepo, auditRepo, resultRepo := tt.setupMocks()
			service := NewTaskServiceWithRepos(taskRepo, auditRepo, resultRepo)

			err := service.PublishFailure(tt.taskID, tt.createdBy, tt.processedBy, tt.errorMsg)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTaskService_ConsumeResult(t *testing.T) {
	tests := []struct {
		name       string
		userID     uint
		wantErr    bool
		setupMocks func() (*MockTaskRepository, *MockTaskAuditRepository, *MockResultRepository)
	}{
		{
			name:    "no results available",
			userID:  1,
			wantErr: true,
			setupMocks: func() (*MockTaskRepository, *MockTaskAuditRepository, *MockResultRepository) {
				resultRepo := &MockResultRepository{
					FindOldestUnconsumedResultByUserIDFunc: func(userID uint) (*database.Result, error) {
						return nil, gorm.ErrRecordNotFound
					},
				}
				return &MockTaskRepository{}, &MockTaskAuditRepository{}, resultRepo
			},
		},
		{
			name:    "success",
			userID:  1,
			wantErr: false,
			setupMocks: func() (*MockTaskRepository, *MockTaskAuditRepository, *MockResultRepository) {
				resultRepo := &MockResultRepository{
					FindOldestUnconsumedResultByUserIDFunc: func(userID uint) (*database.Result, error) {
						return &database.Result{
							ID:        1,
							TaskID:    123,
							CreatedBy: 1,
							Result:    `{"result":"success"}`,
						}, nil
					},
					MarkResultAsConsumedFunc: func(resultID uint) error {
						return nil
					},
				}
				return &MockTaskRepository{}, &MockTaskAuditRepository{}, resultRepo
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskRepo, auditRepo, resultRepo := tt.setupMocks()
			service := NewTaskServiceWithRepos(taskRepo, auditRepo, resultRepo)

			result, err := service.ConsumeResult(tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestTaskService_ReclaimStaleTasks(t *testing.T) {

	config.App = &config.Config{
		Task: config.TaskConfig{
			TimeoutSeconds: 300,
			MaxRetries:     3,
		},
	}

	tests := []struct {
		name          string
		wantReclaimed int
		setupMocks    func() (*MockTaskRepository, *MockTaskAuditRepository, *MockResultRepository)
	}{
		{
			name:          "no stale tasks",
			wantReclaimed: 0,
			setupMocks: func() (*MockTaskRepository, *MockTaskAuditRepository, *MockResultRepository) {
				auditRepo := &MockTaskAuditRepository{
					FindStaleTasksFunc: func(timeoutDuration time.Duration) ([]*database.TaskAudit, error) {
						return []*database.TaskAudit{}, nil
					},
				}
				return &MockTaskRepository{}, auditRepo, &MockResultRepository{}
			},
		},
		{
			name:          "reclaim stale tasks",
			wantReclaimed: 2,
			setupMocks: func() (*MockTaskRepository, *MockTaskAuditRepository, *MockResultRepository) {
				auditRepo := &MockTaskAuditRepository{
					FindStaleTasksFunc: func(timeoutDuration time.Duration) ([]*database.TaskAudit, error) {
						return []*database.TaskAudit{
							{TaskID: 1, RetryCount: 1, Task: database.Task{ID: 1, CreatedBy: 1}},
							{TaskID: 2, RetryCount: 0, Task: database.Task{ID: 2, CreatedBy: 1}},
						}, nil
					},
					ReclaimStaleTaskFunc: func(taskID uint, errorMsg string) error {
						return nil
					},
				}
				return &MockTaskRepository{}, auditRepo, &MockResultRepository{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskRepo, auditRepo, resultRepo := tt.setupMocks()
			service := NewTaskServiceWithRepos(taskRepo, auditRepo, resultRepo)

			reclaimed, err := service.ReclaimStaleTasks()

			assert.NoError(t, err)
			assert.Equal(t, tt.wantReclaimed, reclaimed)
		})
	}
}
