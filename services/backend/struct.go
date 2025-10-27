package main

import (
	"database/sql"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type Todo struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Server struct {
	db *sql.DB
	tracer trace.Tracer
	logger *slog.Logger
}

type DateRange struct {
    Start time.Time `json:"start"`
    End   time.Time `json:"end"`
}

type GroupedTodos struct {
    Date   string `json:"date"`
    Todos  []Todo `json:"todos"`
}
