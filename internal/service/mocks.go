package service

import (
	"time"

	"rainchanel.com/internal/database"
)

type MockUserRepository struct {
	FindByUsernameFunc func(username string) (*database.User, error)
	CreateFunc         func(user *database.User) error
}

func (m *MockUserRepository) FindByUsername(username string) (*database.User, error) {
	if m.FindByUsernameFunc != nil {
		return m.FindByUsernameFunc(username)
	}
	return nil, nil
}

func (m *MockUserRepository) Create(user *database.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(user)
	}
	return nil
}

type MockTaskRepository struct {
	CreateTaskFunc   func(task *database.Task) error
	FindTaskByIDFunc func(taskID uint) (*database.Task, error)
}

func (m *MockTaskRepository) CreateTask(task *database.Task) error {
	if m.CreateTaskFunc != nil {
		return m.CreateTaskFunc(task)
	}
	return nil
}

func (m *MockTaskRepository) FindTaskByID(taskID uint) (*database.Task, error) {
	if m.FindTaskByIDFunc != nil {
		return m.FindTaskByIDFunc(taskID)
	}
	return nil, nil
}

type MockTaskAuditRepository struct {
	CreateTaskAuditFunc          func(audit *database.TaskAudit) error
	FindTaskAuditByTaskIDFunc    func(taskID uint) (*database.TaskAudit, error)
	UpdateTaskAuditStatusFunc    func(taskID uint, status database.TaskStatus) error
	UpdateTaskAuditConsumedFunc  func(taskID uint) error
	UpdateTaskAuditCompletedFunc func(taskID uint, processedBy uint) error
	FindAndClaimPendingTaskFunc  func() (*database.TaskAudit, error)
	FindStaleTasksFunc           func(timeoutDuration time.Duration) ([]*database.TaskAudit, error)
	ReclaimStaleTaskFunc         func(taskID uint, errorMsg string) error
	UpdateTaskFailedFunc         func(taskID uint, errorMsg string) error
	GetTaskStatisticsFunc        func() (map[string]int64, error)
	GetEnhancedStatisticsFunc    func() (map[string]interface{}, error)
	FindTasksWithPaginationFunc  func(limit, offset int, status *database.TaskStatus) ([]*database.TaskAudit, int64, error)
	GetRecentActivityFunc        func(hours int) (map[string]int64, error)
	GetErrorBreakdownFunc        func(limit int) ([]map[string]interface{}, error)
	GetUserStatisticsFunc        func(userID uint) (map[string]int64, error)
	GetUserEnhancedStatisticsFunc func(userID uint) (map[string]interface{}, error)
	FindUserTasksWithPaginationFunc func(userID uint, limit, offset int, status *database.TaskStatus) ([]*database.TaskAudit, int64, error)
	GetUserRecentActivityFunc    func(userID uint, hours int) (map[string]int64, error)
	GetUserErrorBreakdownFunc    func(userID uint, limit int) ([]map[string]interface{}, error)
}

func (m *MockTaskAuditRepository) CreateTaskAudit(audit *database.TaskAudit) error {
	if m.CreateTaskAuditFunc != nil {
		return m.CreateTaskAuditFunc(audit)
	}
	return nil
}

func (m *MockTaskAuditRepository) FindTaskAuditByTaskID(taskID uint) (*database.TaskAudit, error) {
	if m.FindTaskAuditByTaskIDFunc != nil {
		return m.FindTaskAuditByTaskIDFunc(taskID)
	}
	return nil, nil
}

func (m *MockTaskAuditRepository) UpdateTaskAuditStatus(taskID uint, status database.TaskStatus) error {
	if m.UpdateTaskAuditStatusFunc != nil {
		return m.UpdateTaskAuditStatusFunc(taskID, status)
	}
	return nil
}

func (m *MockTaskAuditRepository) UpdateTaskAuditConsumed(taskID uint) error {
	if m.UpdateTaskAuditConsumedFunc != nil {
		return m.UpdateTaskAuditConsumedFunc(taskID)
	}
	return nil
}

func (m *MockTaskAuditRepository) UpdateTaskAuditCompleted(taskID uint, processedBy uint) error {
	if m.UpdateTaskAuditCompletedFunc != nil {
		return m.UpdateTaskAuditCompletedFunc(taskID, processedBy)
	}
	return nil
}

func (m *MockTaskAuditRepository) FindAndClaimPendingTask() (*database.TaskAudit, error) {
	if m.FindAndClaimPendingTaskFunc != nil {
		return m.FindAndClaimPendingTaskFunc()
	}
	return nil, nil
}

func (m *MockTaskAuditRepository) FindStaleTasks(timeoutDuration time.Duration) ([]*database.TaskAudit, error) {
	if m.FindStaleTasksFunc != nil {
		return m.FindStaleTasksFunc(timeoutDuration)
	}
	return nil, nil
}

func (m *MockTaskAuditRepository) ReclaimStaleTask(taskID uint, errorMsg string) error {
	if m.ReclaimStaleTaskFunc != nil {
		return m.ReclaimStaleTaskFunc(taskID, errorMsg)
	}
	return nil
}

func (m *MockTaskAuditRepository) UpdateTaskFailed(taskID uint, errorMsg string) error {
	if m.UpdateTaskFailedFunc != nil {
		return m.UpdateTaskFailedFunc(taskID, errorMsg)
	}
	return nil
}

func (m *MockTaskAuditRepository) GetTaskStatistics() (map[string]int64, error) {
	if m.GetTaskStatisticsFunc != nil {
		return m.GetTaskStatisticsFunc()
	}
	return nil, nil
}

func (m *MockTaskAuditRepository) GetEnhancedStatistics() (map[string]interface{}, error) {
	if m.GetEnhancedStatisticsFunc != nil {
		return m.GetEnhancedStatisticsFunc()
	}
	return nil, nil
}

func (m *MockTaskAuditRepository) FindTasksWithPagination(limit, offset int, status *database.TaskStatus) ([]*database.TaskAudit, int64, error) {
	if m.FindTasksWithPaginationFunc != nil {
		return m.FindTasksWithPaginationFunc(limit, offset, status)
	}
	return nil, 0, nil
}

func (m *MockTaskAuditRepository) GetRecentActivity(hours int) (map[string]int64, error) {
	if m.GetRecentActivityFunc != nil {
		return m.GetRecentActivityFunc(hours)
	}
	return nil, nil
}

func (m *MockTaskAuditRepository) GetErrorBreakdown(limit int) ([]map[string]interface{}, error) {
	if m.GetErrorBreakdownFunc != nil {
		return m.GetErrorBreakdownFunc(limit)
	}
	return nil, nil
}

func (m *MockTaskAuditRepository) GetUserStatistics(userID uint) (map[string]int64, error) {
	if m.GetUserStatisticsFunc != nil {
		return m.GetUserStatisticsFunc(userID)
	}
	return nil, nil
}

func (m *MockTaskAuditRepository) GetUserEnhancedStatistics(userID uint) (map[string]interface{}, error) {
	if m.GetUserEnhancedStatisticsFunc != nil {
		return m.GetUserEnhancedStatisticsFunc(userID)
	}
	return nil, nil
}

func (m *MockTaskAuditRepository) FindUserTasksWithPagination(userID uint, limit, offset int, status *database.TaskStatus) ([]*database.TaskAudit, int64, error) {
	if m.FindUserTasksWithPaginationFunc != nil {
		return m.FindUserTasksWithPaginationFunc(userID, limit, offset, status)
	}
	return nil, 0, nil
}

func (m *MockTaskAuditRepository) GetUserRecentActivity(userID uint, hours int) (map[string]int64, error) {
	if m.GetUserRecentActivityFunc != nil {
		return m.GetUserRecentActivityFunc(userID, hours)
	}
	return nil, nil
}

func (m *MockTaskAuditRepository) GetUserErrorBreakdown(userID uint, limit int) ([]map[string]interface{}, error) {
	if m.GetUserErrorBreakdownFunc != nil {
		return m.GetUserErrorBreakdownFunc(userID, limit)
	}
	return nil, nil
}

type MockResultRepository struct {
	CreateResultFunc                       func(result *database.Result) error
	FindResultByTaskIDFunc                 func(taskID uint) (*database.Result, error)
	FindResultsByUserIDFunc                func(userID uint) ([]database.Result, error)
	FindResultByIDFunc                     func(resultID uint) (*database.Result, error)
	FindOldestUnconsumedResultByUserIDFunc func(userID uint) (*database.Result, error)
	MarkResultAsConsumedFunc               func(resultID uint) error
}

func (m *MockResultRepository) CreateResult(result *database.Result) error {
	if m.CreateResultFunc != nil {
		return m.CreateResultFunc(result)
	}
	return nil
}

func (m *MockResultRepository) FindResultByTaskID(taskID uint) (*database.Result, error) {
	if m.FindResultByTaskIDFunc != nil {
		return m.FindResultByTaskIDFunc(taskID)
	}
	return nil, nil
}

func (m *MockResultRepository) FindResultsByUserID(userID uint) ([]database.Result, error) {
	if m.FindResultsByUserIDFunc != nil {
		return m.FindResultsByUserIDFunc(userID)
	}
	return nil, nil
}

func (m *MockResultRepository) FindResultByID(resultID uint) (*database.Result, error) {
	if m.FindResultByIDFunc != nil {
		return m.FindResultByIDFunc(resultID)
	}
	return nil, nil
}

func (m *MockResultRepository) FindOldestUnconsumedResultByUserID(userID uint) (*database.Result, error) {
	if m.FindOldestUnconsumedResultByUserIDFunc != nil {
		return m.FindOldestUnconsumedResultByUserIDFunc(userID)
	}
	return nil, nil
}

func (m *MockResultRepository) MarkResultAsConsumed(resultID uint) error {
	if m.MarkResultAsConsumedFunc != nil {
		return m.MarkResultAsConsumedFunc(resultID)
	}
	return nil
}
