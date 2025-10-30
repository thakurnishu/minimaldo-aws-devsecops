package handler

import (
	"database/sql"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

type Server struct {
	DB     *sql.DB
	Tracer trace.Tracer
	Logger *slog.Logger
}
