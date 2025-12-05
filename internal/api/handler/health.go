package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"rainchanel.com/internal/database"
	"rainchanel.com/internal/repository"
)

type HealthHandler struct {
	auditRepo repository.TaskAuditRepository
}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		auditRepo: repository.NewTaskAuditRepository(),
	}
}

func (h *HealthHandler) GetHealth(ctx *gin.Context) {

	sqlDB, err := database.DB.DB()
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "Database connection unavailable",
		})
		return
	}

	if err := sqlDB.Ping(); err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "Database ping failed",
		})
		return
	}

	stats, err := h.auditRepo.GetTaskStatistics()
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "degraded",
			"error":  "Failed to get queue statistics",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"queue": gin.H{
			"pending":    stats["pending"],
			"processing": stats["processing"],
			"completed":  stats["completed"],
			"failed":     stats["failed"],
		},
	})
}
