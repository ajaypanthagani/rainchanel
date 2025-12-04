package response

import "rainchanel.com/internal/dto"

type PublishTaskResponse struct {
	TaskID uint `json:"task_id"`
}

type ConsumeTaskResponse struct {
	Task dto.Task `json:"task"`
}

type PublishResultResponse struct {
	Message string `json:"message"`
}

type ConsumeResultResponse struct {
	Result dto.Result `json:"result"`
}
