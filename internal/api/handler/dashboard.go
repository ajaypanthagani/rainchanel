package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"rainchanel.com/internal/database"
	"rainchanel.com/internal/repository"
)

type DashboardHandler struct {
	auditRepo repository.TaskAuditRepository
}

func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{
		auditRepo: repository.NewTaskAuditRepository(),
	}
}

func (h *DashboardHandler) GetDashboard(ctx *gin.Context) {

	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	stats, err := h.auditRepo.GetUserEnhancedStatistics(userIDUint)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get statistics",
		})
		return
	}

	activity, err := h.auditRepo.GetUserRecentActivity(userIDUint, 24)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get recent activity",
		})
		return
	}
	stats["recent_activity_24h"] = activity

	errorBreakdown, err := h.auditRepo.GetUserErrorBreakdown(userIDUint, 10)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get error breakdown",
		})
		return
	}
	stats["error_breakdown"] = errorBreakdown

	ctx.JSON(http.StatusOK, stats)
}

func (h *DashboardHandler) GetTasks(ctx *gin.Context) {

	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	limitStr := ctx.DefaultQuery("limit", "50")
	offsetStr := ctx.DefaultQuery("offset", "0")
	statusStr := ctx.Query("status")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	var status *database.TaskStatus
	if statusStr != "" {
		s := database.TaskStatus(statusStr)
		status = &s
	}

	tasks, total, err := h.auditRepo.FindUserTasksWithPagination(userIDUint, limit, offset, status)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get tasks",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"tasks":  tasks,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *DashboardHandler) GetTaskDetail(ctx *gin.Context) {

	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	taskIDStr := ctx.Param("id")
	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid task ID",
		})
		return
	}

	audit, err := h.auditRepo.FindTaskAuditByTaskID(uint(taskID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "Task not found",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get task",
		})
		return
	}

	if audit.Task.CreatedBy != userIDUint {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied: task does not belong to user",
		})
		return
	}

	ctx.JSON(http.StatusOK, audit)
}
