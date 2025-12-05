package request

import "rainchanel.com/internal/dto"

type PublishTaskRequest struct {
	Task dto.Task `json:"task" binding:"required"`
}

type PublishResultRequest struct {
	TaskID    uint   `json:"task_id" binding:"required"`
	Result    any    `json:"result" binding:"required"`
	CreatedBy uint   `json:"created_by" binding:"required"`
}

type PublishFailureRequest struct {
	TaskID    uint   `json:"task_id" binding:"required"`
	ErrorMsg  string `json:"error_msg" binding:"required"`
	CreatedBy uint   `json:"created_by" binding:"required"`
}
