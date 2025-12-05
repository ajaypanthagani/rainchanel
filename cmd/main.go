package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"rainchanel.com/internal/api/handler"
	"rainchanel.com/internal/config"
	"rainchanel.com/internal/database"
	"rainchanel.com/internal/middleware"
	"rainchanel.com/internal/service"
)

func startServer() {

	if os.Getenv("LOG_FORMAT") == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)

	if err := config.Load(); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if err := database.Init(config.App.Database); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	taskService := service.NewTaskService()
	authService := service.NewAuthService()

	taskHandler := handler.NewTaskHandler(taskService)
	authHandler := handler.NewAuthHandler(authService)
	metricsHandler := handler.NewMetricsHandler()
	healthHandler := handler.NewHealthHandler()
	dashboardHandler := handler.NewDashboardHandler()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	staleTaskService := service.NewStaleTaskService(taskService)
	go staleTaskService.Start(ctx)

	r := gin.Default()

	r.Static("/static", "./web/static")
	r.GET("/", func(ctx *gin.Context) {
		ctx.File("./web/static/index.html")
	})
	r.GET("/login.html", func(ctx *gin.Context) {
		ctx.File("./web/static/login.html")
	})

	r.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"message": "pong"})
	})
	r.GET("/health", healthHandler.GetHealth)
	r.GET("/metrics", metricsHandler.GetMetrics)

	dashboardAPI := r.Group("/api")
	dashboardAPI.Use(middleware.AuthMiddleware())
	{
		dashboardAPI.GET("/dashboard", dashboardHandler.GetDashboard)
		dashboardAPI.GET("/tasks", dashboardHandler.GetTasks)
		dashboardAPI.GET("/tasks/:id", dashboardHandler.GetTaskDetail)
	}

	r.POST("/register", authHandler.Register)
	r.POST("/login", authHandler.Login)

	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/tasks", taskHandler.PublishTask)
		protected.GET("/tasks", taskHandler.ConsumeTask)
		protected.POST("/results", taskHandler.PublishResult)
		protected.POST("/failures", taskHandler.PublishFailure)
		protected.GET("/results", taskHandler.ConsumeResult)
	}

	addr := fmt.Sprintf(":%d", config.App.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		log.Printf("Server started on port %d", config.App.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	fmt.Println("Shutting down...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

func main() {
	startServer()
}
