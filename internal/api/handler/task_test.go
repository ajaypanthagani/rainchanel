package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"rainchanel.com/internal/api/request"
	"rainchanel.com/internal/api/response"
	"rainchanel.com/internal/dto"
	"rainchanel.com/internal/service"
)

type MockTaskService struct {
	PublishTaskFunc       func(task dto.Task, createdBy uint) (uint, error)
	ConsumeTaskFunc       func() (*dto.Task, error)
	PublishResultFunc     func(taskID uint, createdBy uint, processedBy uint, result string) error
	PublishFailureFunc    func(taskID uint, createdBy uint, processedBy uint, errorMsg string) error
	ConsumeResultFunc     func(userID uint) (*dto.Result, error)
	ReclaimStaleTasksFunc func() (int, error)
}

func (m *MockTaskService) PublishTask(task dto.Task, createdBy uint) (uint, error) {
	if m.PublishTaskFunc != nil {
		return m.PublishTaskFunc(task, createdBy)
	}
	return 0, nil
}

func (m *MockTaskService) ConsumeTask() (*dto.Task, error) {
	if m.ConsumeTaskFunc != nil {
		return m.ConsumeTaskFunc()
	}
	return nil, nil
}

func (m *MockTaskService) PublishResult(taskID uint, createdBy uint, processedBy uint, result string) error {
	if m.PublishResultFunc != nil {
		return m.PublishResultFunc(taskID, createdBy, processedBy, result)
	}
	return nil
}

func (m *MockTaskService) ConsumeResult(userID uint) (*dto.Result, error) {
	if m.ConsumeResultFunc != nil {
		return m.ConsumeResultFunc(userID)
	}
	return nil, nil
}

func (m *MockTaskService) PublishFailure(taskID uint, createdBy uint, processedBy uint, errorMsg string) error {
	if m.PublishFailureFunc != nil {
		return m.PublishFailureFunc(taskID, createdBy, processedBy, errorMsg)
	}
	return nil
}

func (m *MockTaskService) ReclaimStaleTasks() (int, error) {
	if m.ReclaimStaleTasksFunc != nil {
		return m.ReclaimStaleTasksFunc()
	}
	return 0, nil
}

func TestNewTaskHandler(t *testing.T) {
	mockService := &MockTaskService{}
	handler := NewTaskHandler(mockService)

	if handler == nil {
		t.Error("NewTaskHandler() returned nil")
	}
}

func TestTaskHandler_PublishTask(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    any
		serviceError   error
		serviceTaskID  uint
		wantStatusCode int
	}{
		{
			name: "success",
			requestBody: request.PublishTaskRequest{
				Task: dto.Task{
					ID:         0,
					WasmModule: "base64-module",
					Func:       "testFunc",
					Args:       []string{"arg1"},
				},
			},
			serviceError:   nil,
			serviceTaskID:  123,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			serviceError:   nil,
			serviceTaskID:  0,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "missing task",
			requestBody: map[string]any{
				"not_task": "value",
			},
			serviceError:   nil,
			serviceTaskID:  0,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name: "service error",
			requestBody: request.PublishTaskRequest{
				Task: dto.Task{
					ID:         0,
					WasmModule: "base64-module",
					Func:       "testFunc",
					Args:       []string{"arg1"},
				},
			},
			serviceError:   errors.New("service error"),
			serviceTaskID:  0,
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockTaskService{
				PublishTaskFunc: func(task dto.Task, createdBy uint) (uint, error) {
					return tt.serviceTaskID, tt.serviceError
				},
			}

			handler := NewTaskHandler(mockService)

			router := gin.New()
			router.POST("/tasks", func(c *gin.Context) {
				if tt.wantStatusCode != http.StatusUnauthorized {
					c.Set("user_id", uint(1))
					c.Set("username", "testuser")
				}
				handler.PublishTask(c)
			})

			var bodyBytes []byte
			var err error
			if tt.name == "invalid JSON" {
				bodyBytes = []byte("invalid json")
			} else {
				bodyBytes, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req, _ := http.NewRequest("POST", "/tasks", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantStatusCode == http.StatusOK {
				var resp response.Response
				err = json.Unmarshal(w.Body.Bytes(), &resp)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				publishResp, ok := resp.Data.(map[string]any)
				if !ok {
					t.Error("Response data is not a map")
				} else {
					var taskID uint
					switch v := publishResp["task_id"].(type) {
					case float64:
						taskID = uint(v)
					case uint:
						taskID = v
					case int:
						taskID = uint(v)
					default:
						t.Errorf("task_id is not a number, got %T", v)
						return
					}
					if taskID != tt.serviceTaskID {
						t.Errorf("task_id = %v, want %v", taskID, tt.serviceTaskID)
					}
				}
			}
		})
	}
}

func TestTaskHandler_ConsumeTask(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		serviceTask    *dto.Task
		serviceError   error
		wantStatusCode int
	}{
		{
			name: "success",
			serviceTask: &dto.Task{
				ID:         123,
				WasmModule: "base64-module",
				Func:       "testFunc",
				Args:       []string{"arg1"},
			},
			serviceError:   nil,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "no tasks available",
			serviceTask:    nil,
			serviceError:   service.ErrNoTasksAvailable,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "service error",
			serviceTask:    nil,
			serviceError:   errors.New("service error"),
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockTaskService{
				ConsumeTaskFunc: func() (*dto.Task, error) {
					return tt.serviceTask, tt.serviceError
				},
			}

			handler := NewTaskHandler(mockService)

			router := gin.New()
			router.GET("/tasks", handler.ConsumeTask)

			req, _ := http.NewRequest("GET", "/tasks", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantStatusCode == http.StatusOK {
				var resp response.Response
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				consumeResp, ok := resp.Data.(map[string]any)
				if !ok {
					t.Error("Response data is not a map")
				} else {
					taskData, ok := consumeResp["task"].(map[string]any)
					if !ok {
						t.Error("task is not a map")
					} else {
						var id uint
						switch v := taskData["id"].(type) {
						case float64:
							id = uint(v)
						case uint:
							id = v
						case int:
							id = uint(v)
						default:
							t.Errorf("task.id is not a number, got %T", v)
							return
						}
						if id != tt.serviceTask.ID {
							t.Errorf("task.id = %v, want %v", id, tt.serviceTask.ID)
						}
					}
				}
			}
		})
	}
}

func TestTaskHandler_PublishResult(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		serviceError   error
		wantStatusCode int
	}{
		{
			name: "success",
			requestBody: map[string]interface{}{
				"task_id":    123,
				"created_by": 1,
				"result":     "success",
			},
			serviceError:   nil,
			wantStatusCode: http.StatusOK,
		},
		{
			name: "missing task_id in body",
			requestBody: map[string]interface{}{
				"result": "success",
			},
			serviceError:   nil,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "missing result in body",
			requestBody: map[string]interface{}{
				"task_id":    123,
				"created_by": 1,
			},
			serviceError:   nil,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "service error",
			requestBody: map[string]interface{}{
				"task_id":    123,
				"created_by": 1,
				"result":     "success",
			},
			serviceError:   errors.New("service error"),
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockTaskService{
				PublishResultFunc: func(taskID uint, createdBy uint, processedBy uint, result string) error {
					return tt.serviceError
				},
			}

			handler := NewTaskHandler(mockService)

			router := gin.New()
			router.POST("/results", func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("username", "testuser")
				handler.PublishResult(c)
			})

			bodyBytes, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			req, _ := http.NewRequest("POST", "/results", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatusCode {
				t.Logf("Response body: %s", w.Body.String())
			}

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantStatusCode == http.StatusOK {
				var resp response.Response
				err = json.Unmarshal(w.Body.Bytes(), &resp)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				var message string
				if completeResp, ok := resp.Data.(map[string]any); ok {
					msg, ok := completeResp["message"].(string)
					if !ok {
						t.Error("message is not a string in map")
						return
					}
					message = msg
				} else if publishResp, ok := resp.Data.(response.PublishResultResponse); ok {
					message = publishResp.Message
				} else {
					dataBytes, _ := json.Marshal(resp.Data)
					var publishResp response.PublishResultResponse
					if err := json.Unmarshal(dataBytes, &publishResp); err == nil {
						message = publishResp.Message
					} else {
						t.Errorf("Response data is not a map or PublishResultResponse: %T", resp.Data)
						return
					}
				}

				if message != "Result published successfully" {
					t.Errorf("message = %v, want 'Result published successfully'", message)
				}
			}
		})
	}
}

func TestTaskHandler_PublishResult_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := &MockTaskService{}
	handler := NewTaskHandler(mockService)

	router := gin.New()
	router.POST("/results", func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("username", "testuser")
		handler.PublishResult(c)
	})

	req, _ := http.NewRequest("POST", "/results", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTaskHandler_PublishFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    any
		serviceError   error
		wantStatusCode int
		setupAuth      func(*gin.Context)
	}{
		{
			name: "success",
			requestBody: request.PublishFailureRequest{
				TaskID:    123,
				CreatedBy: 1,
				ErrorMsg:  "execution failed",
			},
			serviceError:   nil,
			wantStatusCode: http.StatusOK,
			setupAuth: func(c *gin.Context) {
				c.Set("user_id", uint(2))
				c.Set("username", "worker")
			},
		},
		{
			name: "task not found",
			requestBody: request.PublishFailureRequest{
				TaskID:    999,
				CreatedBy: 1,
				ErrorMsg:  "execution failed",
			},
			serviceError:   service.ErrTaskNotFound,
			wantStatusCode: http.StatusNotFound,
			setupAuth: func(c *gin.Context) {
				c.Set("user_id", uint(2))
				c.Set("username", "worker")
			},
		},
		{
			name: "invalid created_by",
			requestBody: request.PublishFailureRequest{
				TaskID:    123,
				CreatedBy: 2,
				ErrorMsg:  "execution failed",
			},
			serviceError:   service.ErrInvalidCreatedBy,
			wantStatusCode: http.StatusForbidden,
			setupAuth: func(c *gin.Context) {
				c.Set("user_id", uint(2))
				c.Set("username", "worker")
			},
		},
		{
			name: "unauthorized - no user_id",
			requestBody: request.PublishFailureRequest{
				TaskID:    123,
				CreatedBy: 1,
				ErrorMsg:  "execution failed",
			},
			wantStatusCode: http.StatusUnauthorized,
			setupAuth: func(c *gin.Context) {

			},
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			wantStatusCode: http.StatusBadRequest,
			setupAuth: func(c *gin.Context) {
				c.Set("user_id", uint(2))
				c.Set("username", "worker")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockTaskService{
				PublishFailureFunc: func(taskID uint, createdBy uint, processedBy uint, errorMsg string) error {
					return tt.serviceError
				},
			}
			handler := NewTaskHandler(mockService)

			var reqBody []byte
			var err error
			if tt.name == "invalid JSON" {
				reqBody = []byte("invalid json")
			} else {
				reqBody, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			router := gin.New()
			router.POST("/failures", func(c *gin.Context) {
				tt.setupAuth(c)
				handler.PublishFailure(c)
			})

			req, _ := http.NewRequest("POST", "/failures", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantStatusCode == http.StatusOK {
				var resp response.Response
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Nil(t, resp.Error)
				assert.NotNil(t, resp.Data)
			}
		})
	}
}
