package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"rainchanel.com/internal/database"
)

type MockTaskAuditRepositoryForMetrics struct {
	GetTaskStatisticsFunc func() (map[string]int64, error)
}

func (m *MockTaskAuditRepositoryForMetrics) GetTaskStatistics() (map[string]int64, error) {
	if m.GetTaskStatisticsFunc != nil {
		return m.GetTaskStatisticsFunc()
	}
	return nil, nil
}

func (m *MockTaskAuditRepositoryForMetrics) CreateTaskAudit(audit *database.TaskAudit) error {
	return nil
}
func (m *MockTaskAuditRepositoryForMetrics) FindTaskAuditByTaskID(taskID uint) (*database.TaskAudit, error) {
	return nil, nil
}
func (m *MockTaskAuditRepositoryForMetrics) UpdateTaskAuditStatus(taskID uint, status database.TaskStatus) error {
	return nil
}
func (m *MockTaskAuditRepositoryForMetrics) UpdateTaskAuditConsumed(taskID uint) error { return nil }
func (m *MockTaskAuditRepositoryForMetrics) UpdateTaskAuditCompleted(taskID uint, processedBy uint) error {
	return nil
}
func (m *MockTaskAuditRepositoryForMetrics) FindAndClaimPendingTask() (*database.TaskAudit, error) {
	return nil, nil
}
func (m *MockTaskAuditRepositoryForMetrics) FindStaleTasks(timeoutDuration time.Duration) ([]*database.TaskAudit, error) {
	return nil, nil
}
func (m *MockTaskAuditRepositoryForMetrics) ReclaimStaleTask(taskID uint, errorMsg string) error {
	return nil
}
func (m *MockTaskAuditRepositoryForMetrics) UpdateTaskFailed(taskID uint, errorMsg string) error {
	return nil
}

func TestMetricsHandler_GetMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupMocks     func() *MockTaskAuditRepositoryForMetrics
		expectedStatus int
		expectedBody   []string
	}{
		{
			name: "success with statistics",
			setupMocks: func() *MockTaskAuditRepositoryForMetrics {
				return &MockTaskAuditRepositoryForMetrics{
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
			expectedStatus: http.StatusOK,
			expectedBody: []string{
				"# HELP rainchanel_tasks_total Total number of tasks by status",
				"# TYPE rainchanel_tasks_total gauge",
				`rainchanel_tasks_total{status="pending"}`,
				`rainchanel_tasks_total{status="processing"}`,
				`rainchanel_tasks_total{status="completed"}`,
				`rainchanel_tasks_total{status="failed"}`,
			},
		},
		{
			name: "error getting statistics",
			setupMocks: func() *MockTaskAuditRepositoryForMetrics {
				return &MockTaskAuditRepositoryForMetrics{
					GetTaskStatisticsFunc: func() (map[string]int64, error) {
						return nil, assert.AnError
					},
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: []string{
				"error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &MetricsHandler{
				auditRepo: tt.setupMocks(),
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/metrics", nil)

			handler.GetMetrics(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			body := w.Body.String()
			for _, expectedLine := range tt.expectedBody {
				assert.Contains(t, body, expectedLine)
			}

			if tt.expectedStatus == http.StatusOK {
				assert.Contains(t, body, "# HELP")
				assert.Contains(t, body, "# TYPE")
				assert.Contains(t, body, "rainchanel_tasks_total")
			}
		})
	}
}

func TestMetricsHandler_GetMetrics_PrometheusFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &MetricsHandler{
		auditRepo: &MockTaskAuditRepositoryForMetrics{
			GetTaskStatisticsFunc: func() (map[string]int64, error) {
				return map[string]int64{
					"pending":    10,
					"processing": 5,
					"completed":  50,
					"failed":     2,
				}, nil
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/metrics", nil)

	handler.GetMetrics(c)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	assert.Contains(t, body, "# HELP rainchanel_tasks_total Total number of tasks by status")
	assert.Contains(t, body, "# TYPE rainchanel_tasks_total gauge")
	assert.Contains(t, body, `rainchanel_tasks_total{status="pending"} 10`)
	assert.Contains(t, body, `rainchanel_tasks_total{status="processing"} 5`)
	assert.Contains(t, body, `rainchanel_tasks_total{status="completed"} 50`)
	assert.Contains(t, body, `rainchanel_tasks_total{status="failed"} 2`)

	lines := strings.Split(strings.TrimSpace(body), "\n")
	assert.Greater(t, len(lines), 4)
}
