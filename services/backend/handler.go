package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func (s *Server) getTodos(c *gin.Context) {
	ctx, span := s.tracer.Start(c.Request.Context(), "get_tasks")
	defer span.End()

	rows, err := s.db.Query(`
		SELECT id, title, description, completed, created_at, updated_at 
		FROM todos 
		ORDER BY created_at DESC
	`)
	if err != nil {
		logError("query failed", ctx, s.logger, span, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var t Todo
		err := rows.Scan(
			&t.ID,
			&t.Title,
			&t.Description,
			&t.Completed,
			&t.CreatedAt,
			&t.UpdatedAt,
		)
		if err != nil {
			logError("rows scan failed", ctx, s.logger, span, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		todos = append(todos, t)
	}

	s.logger.InfoContext(ctx, "Fetching all tasks",
		slog.Int("total_tasks", len(todos)),
	)
	span.SetAttributes(
		attribute.Int("task.count", len(todos)),
	)

	c.JSON(http.StatusOK, todos)
}

func (s *Server) createTodo(c *gin.Context) {
	ctx, span := s.tracer.Start(c.Request.Context(), "create_task")
	defer span.End()

	var t Todo
	if err := c.BindJSON(&t); err != nil {
		logError("failed to parse json", ctx, s.logger, span, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `
		INSERT INTO todos (title, description, completed) 
		VALUES ($1, $2, $3) 
		RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRow(
		query,
		t.Title,
		t.Description,
		t.Completed,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		logError("row scan failed", ctx, s.logger, span, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.logger.InfoContext(ctx, "created task",
		slog.String("task_title", t.Title),
		slog.Bool("task_creation_completed", true),
	)
	span.SetAttributes(
		attribute.String("task.title", t.Title),
		attribute.Bool("task.creation_completed", true),
	)

	c.JSON(http.StatusOK, t)
}

func (s *Server) updateTodo(c *gin.Context) {
	ctx, span := s.tracer.Start(c.Request.Context(), "update_todo")
	defer span.End()

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logError("invalid id", ctx, s.logger, span,err, 
			slog.String("task_id", idStr),
		)

		c.JSON(http.StatusBadRequest, gin.H{"error": "Invaild ID"})
		return
	}

	s.logger.InfoContext(ctx, "todo to update", 
		slog.String("todo_id", idStr),
	)
	span.SetAttributes(
		attribute.String("todo.id", idStr),
	)

	var t Todo
	if err := c.BindJSON(&t); err != nil {
		logError("failed to parse json", ctx, s.logger, span, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	query := `
		UPDATE todos 
		SET title = $1, description = $2, completed = $3
		WHERE id = $4
		RETURNING id, title, description, completed, created_at, updated_at
	`

	err = s.db.QueryRow(
		query,
		t.Title,
		t.Description,
		t.Completed,
		id,
	).Scan(&t.ID, &t.Title, &t.Description, &t.Completed, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			logError("task not found", ctx, s.logger, span, err, 
				slog.Int("task_id", id),
			)
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		logError("row scan failed", ctx, s.logger, span, err)

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.logger.InfoContext(ctx, "task updated",
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

func (s *Server) deleteTodo(c *gin.Context) {
	ctx, span := s.tracer.Start(c.Request.Context(), "delete_task")
	defer span.End()

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logError("invalid id", ctx, s.logger, span, err,
			slog.String("task_id", idStr),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invaild ID"})
		return
	}

	result, err := s.db.Exec("DELETE FROM todos WHERE id = $1", id)
	if err != nil {
		logError("query execution failed", ctx, s.logger, span, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logError("affected rows check failed", ctx, s.logger, span, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if rowsAffected == 0 {
		s.logger.WarnContext(ctx, "task not found",
			slog.Int("task_id", id),
		)
		span.SetStatus(codes.Error, "task not found")

		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	s.logger.InfoContext(ctx, "task delete",
		slog.Int("task_id", id),
		slog.Bool("task_deletion_completed", true),
	)
	span.SetAttributes(
		attribute.Int("task.id", id),
		attribute.Bool("task.deletion_completed", true),
	)

	c.Status(http.StatusNoContent)
}

func (s *Server) healthCheck(c *gin.Context) {
	timeNow := time.Now().Format(time.RFC3339)
	response := map[string]string{
		"status": "healthy",
		"time":   timeNow,
	}
	c.JSON(http.StatusOK, response)
}

func (s *Server) getTodosByDate(c *gin.Context) {
	ctx, span := s.tracer.Start(c.Request.Context(), "get_tasks_by_date")
	defer span.End()

	rangeType := c.Query("range") // day/week/month
	dateStr := c.Query("date")    // YYYY-MM-DD format

	s.logger.InfoContext(ctx, "getting tasks by date range",
		slog.String("range_type", rangeType),
		slog.String("date", dateStr),
	)

	span.SetAttributes(
		attribute.String("request.range_type", rangeType),
		attribute.String("request.date", dateStr),
	)

	dataLayout := "2006-01-02"
	// Parse and validate date
	baseDate, err := time.Parse(dataLayout, dateStr)
	if err != nil {
		logError("invalid date format provided", ctx, s.logger, span, err,
			slog.String("date", dateStr),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
		return
	}

	// Calculate date range with 1-year limit
	now := time.Now()
	if baseDate.Before(now.AddDate(-1, 0, 0)) {
		s.logger.WarnContext(ctx, "date range exceeds 1 year limit",
			slog.String("requested_date", dateStr),
			slog.String("limit_date", now.AddDate(-1, 0, 0).Format(dataLayout)),
		)
		span.SetStatus(codes.Error, "date range exceeds 1 year limit")
		span.SetAttributes(attribute.String("error.type", "date_range_exceeded"))

		c.JSON(http.StatusBadRequest, gin.H{"error": "Date range exceeds 1 year limit"})
		return
	}

	// Calculate date range based on range type
	ctx, dateRangeSpan := s.tracer.Start(ctx, "calculate_date_range")
	var start, end time.Time
	switch rangeType {
	case "day":
		start = time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), 0, 0, 0, 0, time.UTC)
		end = start.AddDate(0, 0, 1)
	case "week":
		start = baseDate.AddDate(0, 0, -int(baseDate.Weekday()))
		end = start.AddDate(0, 0, 7)
	case "month":
		start = time.Date(baseDate.Year(), baseDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		end = start.AddDate(0, 1, 0)
	default:
		logError("invalid range type provided", ctx, s.logger, dateRangeSpan, err,
			slog.String("range_type", rangeType),
		)
		dateRangeSpan.End()
		span.SetStatus(codes.Error, "invalid range type")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid range type"})
		return
	}

	// Add calculated date range to span attributes
	dateRangeSpan.SetAttributes(
		attribute.String("date_range.start", start.Format(time.RFC3339)),
		attribute.String("date_range.end", end.Format(time.RFC3339)),
		attribute.String("date_range.type", rangeType),
	)
	dateRangeSpan.End()

	s.logger.InfoContext(ctx, "calculated date range",
		slog.String("range_type", rangeType),
		slog.String("start_date", start.Format(time.RFC3339)),
		slog.String("end_date", end.Format(time.RFC3339)),
	)

	// Query todos within date range
	ctx, querySpan := s.tracer.Start(ctx, "query_tasks_by_date_range")
	querySpan.SetAttributes(
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "todos"),
		attribute.String("db.query.start_date", start.Format(time.RFC3339)),
		attribute.String("db.query.end_date", end.Format(time.RFC3339)),
	)

	// Query todos within date range
	rows, err := s.db.Query(`
			SELECT id, title, description, completed, created_at, updated_at 
			FROM todos 
			WHERE created_at >= $1 AND created_at < $2
			ORDER BY created_at DESC
	`, start, end)

	if err != nil {
		logError("database query failed", ctx, s.logger, querySpan, err,
			slog.String("start_date", start.Format(time.RFC3339)),
			slog.String("end_date", end.Format(time.RFC3339)),
		)
		querySpan.End()
		span.SetStatus(codes.Error, "database query failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	// Scan results
	ctx, scanSpan := s.tracer.Start(ctx, "scan_tasks_results")
	var todos []Todo
	var todoCount int
	for rows.Next() {
		var t Todo
		err := rows.Scan(
			&t.ID,
			&t.Title,
			&t.Description,
			&t.Completed,
			&t.CreatedAt,
			&t.UpdatedAt,
		)
		if err != nil {
			logError("row scan failed", ctx, s.logger, scanSpan, err,
				slog.String("error", err.Error()),
				slog.Int("scanned_count", todoCount),
			)
			scanSpan.End()
			querySpan.End()
			span.SetStatus(codes.Error, "row scanning failed")

			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		todos = append(todos, t)
		todoCount++
	}

	scanSpan.SetAttributes(
		attribute.Int("todos.count", todoCount),
		attribute.String("scan.status", "completed"),
	)
	scanSpan.End()

	querySpan.SetAttributes(
		attribute.Int("db.rows_affected", todoCount),
		attribute.String("db.operation.status", "success"),
	)
	querySpan.End()

	s.logger.InfoContext(ctx, "successfully retrieved tasks",
		slog.Int("task_count", todoCount),
		slog.String("range_type", rangeType),
	)

	// Group by day if weekly/monthly view
	if rangeType != "day" {
		ctx, groupSpan := s.tracer.Start(ctx, "group_tasks_by_date")
		groupSpan.SetAttributes(
			attribute.String("grouping.type", "by_date"),
			attribute.Int("grouping.input_count", len(todos)),
		)

		grouped := make(map[string][]Todo)
		for _, todo := range todos {
			dateKey := todo.CreatedAt.Format(dataLayout)
			grouped[dateKey] = append(grouped[dateKey], todo)
		}

		var result []GroupedTodos
		for date, items := range grouped {
			result = append(result, GroupedTodos{
				Date:  date,
				Todos: items,
			})
		}

		groupSpan.SetAttributes(
			attribute.Int("grouping.output_groups", len(result)),
			attribute.Int("grouping.total_items", len(todos)),
		)
		groupSpan.End()

		s.logger.InfoContext(ctx, "grouped tasks by date",
			slog.Int("group_count", len(result)),
			slog.Int("total_tasks", len(todos)),
			slog.String("range_type", rangeType),
		)

		// Add final response attributes to main span
		span.SetAttributes(
			attribute.Int("response.group_count", len(result)),
			attribute.Int("response.total_todos", len(todos)),
			attribute.String("response.type", "grouped"),
		)
		c.JSON(http.StatusOK, result)
		return
	}
	// Return ungrouped todos for daily view
	s.logger.InfoContext(ctx, "returning daily tasks",
		slog.Int("tasks_count", len(todos)),
		slog.String("date", dateStr),
	)

	// Add final response attributes to main span
	span.SetAttributes(
		attribute.Int("response.task_count", len(todos)),
		attribute.String("response.type", "list"),
		attribute.String("response.status", "success"),
	)
	c.JSON(http.StatusOK, todos)
}

func logError(msg string, ctx context.Context, logger *slog.Logger, span trace.Span, err error, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("error", err.Error()),
	}
	allAttrs := append(baseAttrs, attrs...)
	logger.LogAttrs(ctx, slog.LevelError, msg, allAttrs...)

	span.RecordError(err)
	span.SetStatus(codes.Error, msg)
}
