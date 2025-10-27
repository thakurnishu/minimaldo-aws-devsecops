package main

import (
	"log/slog"

	_ "github.com/lib/pq"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)


func main() {
	cfg := loadConfig()

	// Otel init
	cleanup, logger, tracer, err := InitTelemetry(cfg)
	if err != nil {
		slog.Error("Failed to initialize telemetry", "error", err)
	}

	defer cleanup()

	db := setupDB(cfg)
	defer db.Close()
	server := &Server{
		db: db,
		logger: logger,
		tracer: tracer,
	}

	router := gin.Default()

	// CORS setup
	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{cfg.FrontendURL},
		AllowHeaders: []string{"X-Requested-With", "Content-Type", "Authorization"},
		AllowMethods: []string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS"},
		ExposeHeaders: []string{"Content-Length"},
	}))
	router.Use(TracingMiddleware(cfg.ServiceName))
	router.Use(LoggingMiddleware(logger))

	// Setup routes
	api := router.Group("/api")
	{
		api.GET("/todos", server.getTodos)
		api.POST("/todos", server.createTodo)
		api.PUT("/todos/:id", server.updateTodo)
		api.DELETE("/todos/:id", server.deleteTodo)
		api.GET("/health", server.healthCheck)
		api.GET("/todos/by-date", server.getTodosByDate)
	}
	
	slog.Info("server is listening", "port", cfg.Port)
	router.Run(":"+cfg.Port)
}
