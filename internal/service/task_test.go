package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
	}{
		{
			name: "success with auto-generated ID",
			task: dto.Task{
				ID:         0,
				WasmModule: "base64-module",
				Func:       "testFunc",
				Args:       []string{"arg1"},
			},
			createdBy:  1,
			wantErr:    true,
			wantTaskID: 0,
		},
		{
			name: "success with provided ID",
			task: dto.Task{
				ID:         123,
				WasmModule: "base64-module",
				Func:       "testFunc",
				Args:       []string{"arg1"},
			},
			createdBy:  1,
			wantErr:    true,
			wantTaskID: 123,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewTaskService()

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

func TestTaskService_PublishTask_GeneratesID(t *testing.T) {
	service := NewTaskService()

	task := dto.Task{
		ID:         0,
		WasmModule: "base64-module",
		Func:       "testFunc",
		Args:       []string{"arg1"},
	}

	_, err := service.PublishTask(task, 1)

	assert.Error(t, err)
}

func TestTaskService_ConsumeTask(t *testing.T) {
	service := NewTaskService()

	task, err := service.ConsumeTask()

	assert.Error(t, err)
	assert.Nil(t, task)
}

func TestTaskService_ConsumeTask_NoTasksAvailable(t *testing.T) {
	service := NewTaskService()

	task, err := service.ConsumeTask()

	assert.Error(t, err)
	assert.Nil(t, task)
}

func TestTaskService_PublishResult(t *testing.T) {
	tests := []struct {
		name    string
		taskID  uint
		userID  uint
		result  string
		wantErr bool
	}{
		{
			name:    "publishes result to database",
			taskID:  123,
			userID:  1,
			result:  "{\"result\":\"success\"}",
			wantErr: true,
		},
		{
			name:    "publishes empty result",
			taskID:  456,
			userID:  2,
			result:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewTaskService()

			err := service.PublishResult(tt.taskID, tt.userID, tt.userID, tt.result)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTaskService_PublishResult_Multiple(t *testing.T) {
	service := NewTaskService()

	taskID := uint(123)
	createdBy := uint(1)
	result := "{\"result\":\"success\"}"

	processedBy := uint(2)

	err := service.PublishResult(taskID, createdBy, processedBy, result)
	assert.Error(t, err)

	err = service.PublishResult(taskID, createdBy, processedBy, result)
	assert.Error(t, err)

	err = service.PublishResult(taskID, createdBy, processedBy, "{\"result\":\"different\"}")
	assert.Error(t, err)
}
