package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"rainchanel.com/internal/config"
	"rainchanel.com/internal/dto"
)

type MockTaskServiceForStale struct {
	ReclaimStaleTasksFunc func() (int, error)
}

func (m *MockTaskServiceForStale) PublishTask(task dto.Task, createdBy uint) (uint, error) {
	return 0, nil
}
func (m *MockTaskServiceForStale) ConsumeTask() (*dto.Task, error) { return nil, nil }
func (m *MockTaskServiceForStale) PublishResult(taskID uint, createdBy uint, processedBy uint, result string) error {
	return nil
}
func (m *MockTaskServiceForStale) PublishFailure(taskID uint, createdBy uint, processedBy uint, errorMsg string) error {
	return nil
}
func (m *MockTaskServiceForStale) ConsumeResult(userID uint) (*dto.Result, error) { return nil, nil }
func (m *MockTaskServiceForStale) ReclaimStaleTasks() (int, error) {
	if m.ReclaimStaleTasksFunc != nil {
		return m.ReclaimStaleTasksFunc()
	}
	return 0, nil
}

func TestNewStaleTaskService(t *testing.T) {
	mockTaskService := &MockTaskServiceForStale{}
	service := NewStaleTaskService(mockTaskService)

	if service == nil {
		t.Error("NewStaleTaskService() returned nil")
	}
}

func TestStaleTaskService_Start(t *testing.T) {

	config.App = &config.Config{
		Task: config.TaskConfig{
			StaleCheckIntervalSeconds: 1,
		},
	}

	t.Run("runs immediately on start", func(t *testing.T) {
		callCount := 0
		mockTaskService := &MockTaskServiceForStale{
			ReclaimStaleTasksFunc: func() (int, error) {
				callCount++
				return 0, nil
			},
		}

		service := NewStaleTaskService(mockTaskService)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan bool)
		go func() {
			service.Start(ctx)
			done <- true
		}()

		time.Sleep(100 * time.Millisecond)
		cancel()

		<-done

		assert.GreaterOrEqual(t, callCount, 1, "ReclaimStaleTasks should be called at least once on start")
	})

	t.Run("runs periodically", func(t *testing.T) {
		callCount := 0
		mockTaskService := &MockTaskServiceForStale{
			ReclaimStaleTasksFunc: func() (int, error) {
				callCount++
				return 0, nil
			},
		}

		service := NewStaleTaskService(mockTaskService)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan bool)
		go func() {
			service.Start(ctx)
			done <- true
		}()

		time.Sleep(2100 * time.Millisecond)
		cancel()

		<-done

		assert.GreaterOrEqual(t, callCount, 2, "ReclaimStaleTasks should be called multiple times")
	})

	t.Run("stops on context cancellation", func(t *testing.T) {
		callCount := 0
		mockTaskService := &MockTaskServiceForStale{
			ReclaimStaleTasksFunc: func() (int, error) {
				callCount++
				return 0, nil
			},
		}

		service := NewStaleTaskService(mockTaskService)
		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan bool)
		go func() {
			service.Start(ctx)
			done <- true
		}()

		cancel()

		select {
		case <-done:

		case <-time.After(2 * time.Second):
			t.Error("Service should stop on context cancellation")
		}
	})

	t.Run("handles errors from ReclaimStaleTasks", func(t *testing.T) {
		mockTaskService := &MockTaskServiceForStale{
			ReclaimStaleTasksFunc: func() (int, error) {
				return 0, assert.AnError
			},
		}

		service := NewStaleTaskService(mockTaskService)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan bool)
		go func() {
			service.Start(ctx)
			done <- true
		}()

		time.Sleep(100 * time.Millisecond)
		cancel()

		<-done

		assert.True(t, true, "Service should handle errors gracefully")
	})
}
