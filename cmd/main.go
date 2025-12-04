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
	"rainchanel.com/internal/api/handler"
	"rainchanel.com/internal/config"
	"rainchanel.com/internal/database"
	"rainchanel.com/internal/middleware"
	"rainchanel.com/internal/service"
)

func startServer() {
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

	r := gin.Default()

	r.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	r.POST("/register", authHandler.Register)
	r.POST("/login", authHandler.Login)

	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/tasks", taskHandler.PublishTask)
		protected.GET("/tasks", taskHandler.ConsumeTask)
		protected.POST("/results", taskHandler.PublishResult)
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

func main() {
	startServer()
}
