package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"rainchanel.com/internal/api/request"
	"rainchanel.com/internal/api/response"
	"rainchanel.com/internal/service"
)

type TaskHandler interface {
	PublishTask(*gin.Context)
	ConsumeTask(*gin.Context)
	PublishResult(*gin.Context)
	PublishFailure(*gin.Context)
	ConsumeResult(*gin.Context)
}

type taskHandler struct {
	taskService service.TaskService
}

func NewTaskHandler(taskService service.TaskService) TaskHandler {
	return &taskHandler{
		taskService: taskService,
	}
}

func (h *taskHandler) PublishTask(ctx *gin.Context) {
	var createTaskRequest request.PublishTaskRequest

	if err := ctx.ShouldBindJSON(&createTaskRequest); err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Error: &response.Error{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			},
		})
		return
	}

	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, response.Response{
			Error: &response.Error{
				Code:    http.StatusUnauthorized,
				Message: "User not authenticated",
			},
		})
		return
	}

	taskID, err := h.taskService.PublishTask(createTaskRequest.Task, userID.(uint))

	if err != nil {
		ctx.JSON(500, response.Response{
			Error: &response.Error{
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
			},
		})
		return
	}

	ctx.JSON(200, response.Response{
		Data: response.PublishTaskResponse{
			TaskID: taskID,
		},
	})
}

func (h *taskHandler) ConsumeTask(ctx *gin.Context) {
	task, err := h.taskService.ConsumeTask()

	if err != nil {
		if errors.Is(err, service.ErrNoTasksAvailable) {
			ctx.JSON(http.StatusNotFound, response.Response{
				Error: &response.Error{
					Code:    http.StatusNotFound,
					Message: "No tasks available to consume",
				},
			})
			return
		}

		ctx.JSON(500, response.Response{
			Error: &response.Error{
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
			},
		})
		return
	}

	ctx.JSON(200, response.Response{
		Data: response.ConsumeTaskResponse{
			Task: *task,
		},
	})
}

func (h *taskHandler) PublishResult(ctx *gin.Context) {
	var publishResultRequest request.PublishResultRequest

	if err := ctx.ShouldBindJSON(&publishResultRequest); err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Error: &response.Error{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			},
		})
		return
	}

	processedBy, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, response.Response{
			Error: &response.Error{
				Code:    http.StatusUnauthorized,
				Message: "User not authenticated",
			},
		})
		return
	}

	resultJSON, err := json.Marshal(publishResultRequest.Result)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Error: &response.Error{
				Code:    http.StatusBadRequest,
				Message: "Invalid result format",
			},
		})
		return
	}

	err = h.taskService.PublishResult(
		publishResultRequest.TaskID,
		publishResultRequest.CreatedBy,
		processedBy.(uint),
		string(resultJSON),
	)

	if err != nil {
		if errors.Is(err, service.ErrTaskNotFound) {
			ctx.JSON(http.StatusNotFound, response.Response{
				Error: &response.Error{
					Code:    http.StatusNotFound,
					Message: "Task not found",
				},
			})
			return
		}
		if errors.Is(err, service.ErrInvalidCreatedBy) {
			ctx.JSON(http.StatusForbidden, response.Response{
				Error: &response.Error{
					Code:    http.StatusForbidden,
					Message: "Invalid created_by - does not match task record",
				},
			})
			return
		}

		ctx.JSON(500, response.Response{
			Error: &response.Error{
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
			},
		})
		return
	}

	ctx.JSON(200, response.Response{
		Data: response.PublishResultResponse{
			Message: "Result published successfully",
		},
	})
}

func (h *taskHandler) PublishFailure(ctx *gin.Context) {
	var publishFailureRequest request.PublishFailureRequest

	if err := ctx.ShouldBindJSON(&publishFailureRequest); err != nil {
		ctx.JSON(http.StatusBadRequest, response.Response{
			Error: &response.Error{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			},
		})
		return
	}

	processedBy, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, response.Response{
			Error: &response.Error{
				Code:    http.StatusUnauthorized,
				Message: "User not authenticated",
			},
		})
		return
	}

	err := h.taskService.PublishFailure(
		publishFailureRequest.TaskID,
		publishFailureRequest.CreatedBy,
		processedBy.(uint),
		publishFailureRequest.ErrorMsg,
	)

	if err != nil {
		if errors.Is(err, service.ErrTaskNotFound) {
			ctx.JSON(http.StatusNotFound, response.Response{
				Error: &response.Error{
					Code:    http.StatusNotFound,
					Message: "Task not found",
				},
			})
			return
		}
		if errors.Is(err, service.ErrInvalidCreatedBy) {
			ctx.JSON(http.StatusForbidden, response.Response{
				Error: &response.Error{
					Code:    http.StatusForbidden,
					Message: "Invalid created_by - does not match task record",
				},
			})
			return
		}

		ctx.JSON(500, response.Response{
			Error: &response.Error{
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
			},
		})
		return
	}

	ctx.JSON(200, response.Response{
		Data: response.PublishResultResponse{
			Message: "Failure recorded, task will be retried if retries available",
		},
	})
}

func (h *taskHandler) ConsumeResult(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, response.Response{
			Error: &response.Error{
				Code:    http.StatusUnauthorized,
				Message: "User not authenticated",
			},
		})
		return
	}

	result, err := h.taskService.ConsumeResult(userID.(uint))
	if err != nil {
		if errors.Is(err, service.ErrNoTasksAvailable) {
			ctx.JSON(http.StatusNotFound, response.Response{
				Error: &response.Error{
					Code:    http.StatusNotFound,
					Message: "No results available",
				},
			})
			return
		}

		ctx.JSON(500, response.Response{
			Error: &response.Error{
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
			},
		})
		return
	}

	ctx.JSON(200, response.Response{
		Data: response.ConsumeResultResponse{
			Result: *result,
		},
	})
}
