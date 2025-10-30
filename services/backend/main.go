package main

import (
	"log/slog"

	_ "github.com/lib/pq"
	"github.com/thakurnishu/MinimalDo/config"
	"github.com/thakurnishu/MinimalDo/db"
	"github.com/thakurnishu/MinimalDo/handler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()

	// Otel init
	cleanup, logger, tracer, err := InitTelemetry(cfg)
	if err != nil {
		slog.Error("Failed to initialize telemetry", "error", err)
	}

	defer cleanup()

	db := db.SetupDB(cfg)
	defer db.Close()
	server := &handler.Server{
		DB:     db,
		Logger: logger,
		Tracer: tracer,
	}

	router := gin.Default()

	// CORS setup
	router.Use(cors.New(cors.Config{
		AllowOrigins:  []string{cfg.FrontendURL},
		AllowHeaders:  []string{"X-Requested-With", "Content-Type", "Authorization"},
		AllowMethods:  []string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS"},
		ExposeHeaders: []string{"Content-Length"},
	}))
	router.Use(TracingMiddleware(cfg.ServiceName))
	router.Use(LoggingMiddleware(logger))

	// Setup routes
	api := router.Group("/api")
	{
		api.GET("/todos", server.GetTodos)
		api.POST("/todos", server.CreateTodo)
		api.PUT("/todos/:id", server.UpdateTodo)
		api.DELETE("/todos/:id", server.DeleteTodo)
		api.GET("/health", server.HealthCheck)
		api.GET("/todos/by-date", server.GetTodosByDate)
	}

	slog.Info("server is listening", "port", cfg.Port)
	router.Run(":" + cfg.Port)
}
