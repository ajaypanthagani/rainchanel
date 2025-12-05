package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"rainchanel.com/internal/database"
)

type MockTaskAuditRepositoryForHealth struct {
	GetTaskStatisticsFunc func() (map[string]int64, error)
}

func (m *MockTaskAuditRepositoryForHealth) GetTaskStatistics() (map[string]int64, error) {
	if m.GetTaskStatisticsFunc != nil {
		return m.GetTaskStatisticsFunc()
	}
	return nil, nil
}

func (m *MockTaskAuditRepositoryForHealth) CreateTaskAudit(audit *database.TaskAudit) error {
	return nil
}
func (m *MockTaskAuditRepositoryForHealth) FindTaskAuditByTaskID(taskID uint) (*database.TaskAudit, error) {
	return nil, nil
}
func (m *MockTaskAuditRepositoryForHealth) UpdateTaskAuditStatus(taskID uint, status database.TaskStatus) error {
	return nil
}
func (m *MockTaskAuditRepositoryForHealth) UpdateTaskAuditConsumed(taskID uint) error { return nil }
func (m *MockTaskAuditRepositoryForHealth) UpdateTaskAuditCompleted(taskID uint, processedBy uint) error {
	return nil
}
func (m *MockTaskAuditRepositoryForHealth) FindAndClaimPendingTask() (*database.TaskAudit, error) {
	return nil, nil
}
func (m *MockTaskAuditRepositoryForHealth) FindStaleTasks(timeoutDuration time.Duration) ([]*database.TaskAudit, error) {
	return nil, nil
}
func (m *MockTaskAuditRepositoryForHealth) ReclaimStaleTask(taskID uint, errorMsg string) error {
	return nil
}
func (m *MockTaskAuditRepositoryForHealth) UpdateTaskFailed(taskID uint, errorMsg string) error {
	return nil
}
func (m *MockTaskAuditRepositoryForHealth) GetEnhancedStatistics() (map[string]interface{}, error) {
	return nil, nil
}
func (m *MockTaskAuditRepositoryForHealth) FindTasksWithPagination(limit, offset int, status *database.TaskStatus) ([]*database.TaskAudit, int64, error) {
	return nil, 0, nil
}
func (m *MockTaskAuditRepositoryForHealth) GetRecentActivity(hours int) (map[string]int64, error) {
	return nil, nil
}
func (m *MockTaskAuditRepositoryForHealth) GetErrorBreakdown(limit int) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *MockTaskAuditRepositoryForHealth) GetUserStatistics(userID uint) (map[string]int64, error) {
	return nil, nil
}
func (m *MockTaskAuditRepositoryForHealth) GetUserEnhancedStatistics(userID uint) (map[string]interface{}, error) {
	return nil, nil
}
func (m *MockTaskAuditRepositoryForHealth) FindUserTasksWithPagination(userID uint, limit, offset int, status *database.TaskStatus) ([]*database.TaskAudit, int64, error) {
	return nil, 0, nil
}
func (m *MockTaskAuditRepositoryForHealth) GetUserRecentActivity(userID uint, hours int) (map[string]int64, error) {
	return nil, nil
}
func (m *MockTaskAuditRepositoryForHealth) GetUserErrorBreakdown(userID uint, limit int) ([]map[string]interface{}, error) {
	return nil, nil
}

func TestHealthHandler_GetHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupMocks     func() *MockTaskAuditRepositoryForHealth
		setupDB        func() error
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "healthy with statistics",
			setupMocks: func() *MockTaskAuditRepositoryForHealth {
				return &MockTaskAuditRepositoryForHealth{
					GetTaskStatisticsFunc: func() (map[string]int64, error) {
						return map[string]int64{
							"pending":    5,
							"processing": 2,
							"completed":  100,
							"failed":     3,
						}, nil
					},
				}
			},
			setupDB: func() error {

				return nil
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"status": "healthy",
				"queue": map[string]interface{}{
					"pending":    int64(5),
					"processing": int64(2),
					"completed":  int64(100),
					"failed":     int64(3),
				},
			},
		},
		{
			name: "degraded - statistics error",
			setupMocks: func() *MockTaskAuditRepositoryForHealth {
				return &MockTaskAuditRepositoryForHealth{
					GetTaskStatisticsFunc: func() (map[string]int64, error) {
						return nil, assert.AnError
					},
				}
			},
			setupDB: func() error {
				return nil
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody: map[string]interface{}{
				"status": "degraded",
				"error":  "Failed to get queue statistics",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if database.DB == nil {
				t.Skip("Database not initialized - skipping health check test")
				return
			}

			handler := &HealthHandler{
				auditRepo: tt.setupMocks(),
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

			handler.GetHealth(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectedBody["status"] != nil {
				assert.Equal(t, tt.expectedBody["status"], response["status"])
			}
		})
	}
}
