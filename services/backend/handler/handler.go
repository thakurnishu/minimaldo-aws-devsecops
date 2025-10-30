package handler

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thakurnishu/MinimalDo/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func (s *Server) GetTodos(c *gin.Context) {
	ctx, span := s.Tracer.Start(c.Request.Context(), "get_tasks")
	defer span.End()

	rows, err := s.DB.Query(`
		SELECT id, title, description, completed, created_at, updated_at 
		FROM todos 
		ORDER BY created_at DESC
	`)
	if err != nil {
		logError("query failed", ctx, s.Logger, span, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var todos []types.Todo
	for rows.Next() {
		var t types.Todo
		err := rows.Scan(
			&t.ID,
			&t.Title,
			&t.Description,
			&t.Completed,
			&t.CreatedAt,
			&t.UpdatedAt,
		)
		if err != nil {
			logError("rows scan failed", ctx, s.Logger, span, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		todos = append(todos, t)
	}

	s.Logger.InfoContext(ctx, "Fetching all tasks",
		slog.Int("total_tasks", len(todos)),
	)
	span.SetAttributes(
		attribute.Int("task.count", len(todos)),
	)

	c.JSON(http.StatusOK, todos)
}

func (s *Server) CreateTodo(c *gin.Context) {
	ctx, span := s.Tracer.Start(c.Request.Context(), "create_task")
	defer span.End()

	var t types.Todo
	if err := c.BindJSON(&t); err != nil {
		logError("failed to parse json", ctx, s.Logger, span, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `
		INSERT INTO todos (title, description, completed) 
		VALUES ($1, $2, $3) 
		RETURNING id, created_at, updated_at
	`

	err := s.DB.QueryRow(
		query,
		t.Title,
		t.Description,
		t.Completed,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		logError("row scan failed", ctx, s.Logger, span, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.Logger.InfoContext(ctx, "created task",
		slog.String("task_title", t.Title),
		slog.Bool("task_creation_completed", true),
	)
	span.SetAttributes(
		attribute.String("task.title", t.Title),
		attribute.Bool("task.creation_completed", true),
	)

	c.JSON(http.StatusOK, t)
}

func (s *Server) UpdateTodo(c *gin.Context) {
	ctx, span := s.Tracer.Start(c.Request.Context(), "update_todo")
	defer span.End()

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logError("invalid id", ctx, s.Logger, span, err,
			slog.String("task_id", idStr),
		)

		c.JSON(http.StatusBadRequest, gin.H{"error": "Invaild ID"})
		return
	}

	s.Logger.InfoContext(ctx, "todo to update",
		slog.String("todo_id", idStr),
	)
	span.SetAttributes(
		attribute.String("todo.id", idStr),
	)

	var t types.Todo
	if err := c.BindJSON(&t); err != nil {
		logError("failed to parse json", ctx, s.Logger, span, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	query := `
		UPDATE todos 
		SET title = $1, description = $2, completed = $3
		WHERE id = $4
		RETURNING id, title, description, completed, created_at, updated_at
	`

	err = s.DB.QueryRow(
		query,
		t.Title,
		t.Description,
		t.Completed,
		id,
	).Scan(&t.ID, &t.Title, &t.Description, &t.Completed, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			logError("task not found", ctx, s.Logger, span, err,
				slog.Int("task_id", id),
			)
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		logError("row scan failed", ctx, s.Logger, span, err)

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.Logger.InfoContext(ctx, "task updated",
		slog.Int("task_id", id),
		slog.String("task_title", t.Title),
		slog.Bool("task_updation_completed", true),
	)
	span.SetAttributes(
		attribute.Int("task.id", id),
		attribute.String("task.title", t.Title),
		attribute.Bool("task.updation_completed", true),
	)

	c.JSON(http.StatusOK, t)
}

func (s *Server) DeleteTodo(c *gin.Context) {
	ctx, span := s.Tracer.Start(c.Request.Context(), "delete_task")
	defer span.End()

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logError("invalid id", ctx, s.Logger, span, err,
			slog.String("task_id", idStr),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invaild ID"})
		return
	}

	result, err := s.DB.Exec("DELETE FROM todos WHERE id = $1", id)
	if err != nil {
		logError("query execution failed", ctx, s.Logger, span, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logError("affected rows check failed", ctx, s.Logger, span, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if rowsAffected == 0 {
		s.Logger.WarnContext(ctx, "task not found",
			slog.Int("task_id", id),
		)
		span.SetStatus(codes.Error, "task not found")

		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	s.Logger.InfoContext(ctx, "task delete",
		slog.Int("task_id", id),
		slog.Bool("task_deletion_completed", true),
	)
	span.SetAttributes(
		attribute.Int("task.id", id),
		attribute.Bool("task.deletion_completed", true),
	)

	c.Status(http.StatusNoContent)
}

func (s *Server) HealthCheck(c *gin.Context) {
	timeNow := time.Now().Format(time.RFC3339)
	response := map[string]string{
		"status": "healthy",
		"time":   timeNow,
	}
	c.JSON(http.StatusOK, response)
}

func (s *Server) GetTodosByDate(c *gin.Context) {
	ctx, span := s.Tracer.Start(c.Request.Context(), "handler.get_todos_by_date")
	defer span.End()

	rangeType := c.Query("range") // day, week, month
	dateStr := c.Query("date")    // YYYY-MM-DD format

	s.Logger.InfoContext(ctx, "fetching todos by date range",
		slog.String("range_type", rangeType),
		slog.String("date", dateStr),
	)

	span.SetAttributes(
		attribute.String("param.range_type", rangeType),
		attribute.String("param.date", dateStr),
	)

	// --- Parse date safely ---
	const layout = "2006-01-02"
	baseDate, err := time.Parse(layout, dateStr)
	if err != nil {
		s.respondWithError(c, http.StatusBadRequest, "Invalid date format (expected YYYY-MM-DD)", ctx, span, err)
		return
	}

	// --- Prevent excessively old queries (> 1 year) ---
	if baseDate.Before(time.Now().AddDate(-1, 0, 0)) {
		s.respondWithError(c, http.StatusBadRequest, "Date range exceeds 1 year limit", ctx, span, nil)
		return
	}

	// --- Compute date range ---
	start, end, err := calculateDateRange(baseDate, rangeType)
	if err != nil {
		s.respondWithError(c, http.StatusBadRequest, "Invalid range type (use day|week|month)", ctx, span, err)
		return
	}

	span.SetAttributes(
		attribute.String("range.start", start.Format(time.RFC3339)),
		attribute.String("range.end", end.Format(time.RFC3339)),
	)

	s.Logger.DebugContext(ctx, "calculated date range",
		slog.String("start", start.Format(time.RFC3339)),
		slog.String("end", end.Format(time.RFC3339)),
	)

	// --- Query DB ---
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, title, description, completed, created_at, updated_at
		FROM todos
		WHERE created_at >= $1 AND created_at < $2
		ORDER BY created_at DESC
	`, start, end)
	if err != nil {
		s.respondWithError(c, http.StatusInternalServerError, "Database query failed", ctx, span, err)
		return
	}
	defer rows.Close()

	var todos []types.Todo
	for rows.Next() {
		var t types.Todo
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Completed, &t.CreatedAt, &t.UpdatedAt); err != nil {
			s.respondWithError(c, http.StatusInternalServerError, "Failed to parse database row", ctx, span, err)
			return
		}
		todos = append(todos, t)
	}

	if err = rows.Err(); err != nil {
		s.respondWithError(c, http.StatusInternalServerError, "Database row error", ctx, span, err)
		return
	}

	s.Logger.InfoContext(ctx, "retrieved todos",
		slog.Int("count", len(todos)),
		slog.String("range_type", rangeType),
	)

	// --- Group by date if weekly/monthly ---
	if rangeType != "day" {
		grouped := groupTodosByDate(todos, layout)
		span.SetAttributes(
			attribute.Int("response.group_count", len(grouped)),
			attribute.Int("response.total_todos", len(todos)),
		)
		c.JSON(http.StatusOK, grouped)
		return
	}

	// --- Otherwise, return flat list ---
	c.JSON(http.StatusOK, todos)
}

func logError(msg string, ctx context.Context, Logger *slog.Logger, span trace.Span, err error, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("error", err.Error()),
	}
	allAttrs := append(baseAttrs, attrs...)
	Logger.LogAttrs(ctx, slog.LevelError, msg, allAttrs...)

	span.RecordError(err)
	span.SetStatus(codes.Error, msg)
}

// calculateDateRange returns start and end for day|week|month
func calculateDateRange(baseDate time.Time, rangeType string) (time.Time, time.Time, error) {
	switch rangeType {
	case "day":
		start := time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 0, 1), nil
	case "week":
		start := baseDate.AddDate(0, 0, -int(baseDate.Weekday()))
		return start, start.AddDate(0, 0, 7), nil
	case "month":
		start := time.Date(baseDate.Year(), baseDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0), nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("invalid range type: %s", rangeType)
	}
}

// groupTodosByDate groups todos by creation date (YYYY-MM-DD)
func groupTodosByDate(todos []types.Todo, layout string) []types.GroupedTodos {
	groupedMap := make(map[string][]types.Todo)
	for _, t := range todos {
		key := t.CreatedAt.Format(layout)
		groupedMap[key] = append(groupedMap[key], t)
	}

	var result []types.GroupedTodos
	for date, list := range groupedMap {
		result = append(result, types.GroupedTodos{Date: date, Todos: list})
	}
	return result
}

// respondWithError centralizes error + trace reporting
func (s *Server) respondWithError(
	c *gin.Context,
	code int,
	msg string,
	ctx context.Context,
	span trace.Span,
	err error,
) {
	if err != nil {
		logError(msg, ctx, s.Logger, span, err)
	} else {
		s.Logger.WarnContext(ctx, msg)
	}
	span.SetStatus(codes.Error, msg)
	c.JSON(code, gin.H{"error": msg})
}
