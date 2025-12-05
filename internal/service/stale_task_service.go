package service

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"rainchanel.com/internal/config"
)

type StaleTaskService interface {
	Start(ctx context.Context)
}

type staleTaskService struct {
	taskService TaskService
}

func NewStaleTaskService(taskService TaskService) StaleTaskService {
	return &staleTaskService{
		taskService: taskService,
	}
}

func (s *staleTaskService) Start(ctx context.Context) {
	interval := time.Duration(config.App.Task.StaleCheckIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logrus.WithFields(logrus.Fields{
		"check_interval_seconds": config.App.Task.StaleCheckIntervalSeconds,
	}).Info("Stale task detection service started")

	s.checkAndReclaimStaleTasks()

	for {
		select {
		case <-ctx.Done():
			logrus.Info("Stale task detection service stopped")
			return
		case <-ticker.C:
			s.checkAndReclaimStaleTasks()
		}
	}
}

func (s *staleTaskService) checkAndReclaimStaleTasks() {
	reclaimedCount, err := s.taskService.ReclaimStaleTasks()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Error reclaiming stale tasks")
		return
	}
	if reclaimedCount > 0 {
		logrus.WithFields(logrus.Fields{
			"count": reclaimedCount,
		}).Info("Reclaimed stale tasks")
	}
}
