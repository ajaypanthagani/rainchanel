package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"rainchanel.com/internal/repository"
)

type MetricsHandler struct {
	auditRepo repository.TaskAuditRepository
}

func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{
		auditRepo: repository.NewTaskAuditRepository(),
	}
}

func (h *MetricsHandler) GetMetrics(ctx *gin.Context) {
	stats, err := h.auditRepo.GetTaskStatistics()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get metrics",
		})
		return
	}

	var metrics string
	metrics = "# HELP rainchanel_tasks_total Total number of tasks by status\n"
	metrics += "# TYPE rainchanel_tasks_total gauge\n"

	for status, count := range stats {
		metrics += `rainchanel_tasks_total{status="` + status + `"}` + " " + strconv.FormatInt(count, 10) + "\n"
	}

	ctx.String(http.StatusOK, metrics)
}
