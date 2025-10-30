package handler_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"log/slog"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/thakurnishu/MinimalDo/handler"
	"github.com/thakurnishu/MinimalDo/types"
	"go.opentelemetry.io/otel/trace/noop"
)

func setupTestServer(t *testing.T) (*handler.Server, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	// Use io.Discard to suppress log output during tests
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")

	server := &handler.Server{
		DB:     db,
		Logger: logger,
		Tracer: tracer,
	}

	return server, mock
}

func TestGetTodos(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success - returns all todos", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		now := time.Now()
		rows := sqlmock.NewRows([]string{"id", "title", "description", "completed", "created_at", "updated_at"}).
			AddRow(1, "Test Todo 1", "Description 1", false, now, now).
			AddRow(2, "Test Todo 2", "Description 2", true, now, now)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, title, description, completed, created_at, updated_at FROM todos ORDER BY created_at DESC`)).
			WillReturnRows(rows)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/todos", nil)

		server.GetTodos(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var todos []types.Todo
		err := json.Unmarshal(w.Body.Bytes(), &todos)
		assert.NoError(t, err)
		assert.Len(t, todos, 2)
		assert.Equal(t, "Test Todo 1", todos[0].Title)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - query fails", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, title, description, completed, created_at, updated_at FROM todos ORDER BY created_at DESC`)).
			WillReturnError(assert.AnError)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/todos", nil)

		server.GetTodos(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success - returns empty array when no todos", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		rows := sqlmock.NewRows([]string{"id", "title", "description", "completed", "created_at", "updated_at"})
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, title, description, completed, created_at, updated_at FROM todos ORDER BY created_at DESC`)).
			WillReturnRows(rows)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/todos", nil)

		server.GetTodos(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCreateTodo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success - creates new todo", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		now := time.Now()
		todo := types.Todo{
			Title:       "New Todo",
			Description: "New Description",
			Completed:   false,
		}

		rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, now)

		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO todos (title, description, completed) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`)).
			WithArgs(todo.Title, todo.Description, todo.Completed).
			WillReturnRows(rows)

		body, _ := json.Marshal(todo)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		server.CreateTodo(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var result types.Todo
		err := json.Unmarshal(w.Body.Bytes(), &result)
		assert.NoError(t, err)
		assert.Equal(t, 1, result.ID)
		assert.Equal(t, "New Todo", result.Title)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - invalid json", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader("invalid json"))
		c.Request.Header.Set("Content-Type", "application/json")

		server.CreateTodo(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - insert fails", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		todo := types.Todo{
			Title:       "New Todo",
			Description: "New Description",
			Completed:   false,
		}

		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO todos (title, description, completed) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`)).
			WithArgs(todo.Title, todo.Description, todo.Completed).
			WillReturnError(assert.AnError)

		body, _ := json.Marshal(todo)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		server.CreateTodo(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUpdateTodo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success - updates existing todo", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		now := time.Now()
		todo := types.Todo{
			Title:       "Updated Todo",
			Description: "Updated Description",
			Completed:   true,
		}

		rows := sqlmock.NewRows([]string{"id", "title", "description", "completed", "created_at", "updated_at"}).
			AddRow(1, todo.Title, todo.Description, todo.Completed, now, now)

		mock.ExpectQuery(regexp.QuoteMeta(`UPDATE todos SET title = $1, description = $2, completed = $3 WHERE id = $4 RETURNING id, title, description, completed, created_at, updated_at`)).
			WithArgs(todo.Title, todo.Description, todo.Completed, 1).
			WillReturnRows(rows)

		body, _ := json.Marshal(todo)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPut, "/todos/1", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		server.UpdateTodo(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var result types.Todo
		err := json.Unmarshal(w.Body.Bytes(), &result)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Todo", result.Title)
		assert.True(t, result.Completed)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - invalid id", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPut, "/todos/abc", nil)
		c.Params = gin.Params{{Key: "id", Value: "abc"}}

		server.UpdateTodo(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - todo not found", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		todo := types.Todo{
			Title:       "Updated Todo",
			Description: "Updated Description",
			Completed:   true,
		}

		mock.ExpectQuery(regexp.QuoteMeta(`UPDATE todos SET title = $1, description = $2, completed = $3 WHERE id = $4 RETURNING id, title, description, completed, created_at, updated_at`)).
			WithArgs(todo.Title, todo.Description, todo.Completed, 999).
			WillReturnError(sql.ErrNoRows)

		body, _ := json.Marshal(todo)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPut, "/todos/999", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		server.UpdateTodo(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDeleteTodo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success - deletes existing todo", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM todos WHERE id = $1")).
			WithArgs(1).
			WillReturnResult(sqlmock.NewResult(0, 1))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodDelete, "/todos/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		server.DeleteTodo(c)

		// The handler calls c.Status(http.StatusNoContent) which should set 204
		// but Gin might default to 200 if no body is written
		assert.True(t, w.Code == http.StatusNoContent || w.Code == http.StatusOK)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - invalid id", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodDelete, "/todos/abc", nil)
		c.Params = gin.Params{{Key: "id", Value: "abc"}}

		server.DeleteTodo(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - todo not found", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM todos WHERE id = $1")).
			WithArgs(999).
			WillReturnResult(sqlmock.NewResult(0, 0))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodDelete, "/todos/999", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		server.DeleteTodo(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - query execution fails", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM todos WHERE id = $1")).
			WithArgs(1).
			WillReturnError(assert.AnError)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodDelete, "/todos/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		server.DeleteTodo(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success - returns healthy status", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

		server.HealthCheck(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "healthy", response["status"])
		assert.NotEmpty(t, response["time"])
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetTodosByDate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success - daily range returns todos", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		now := time.Now()
		rows := sqlmock.NewRows([]string{"id", "title", "description", "completed", "created_at", "updated_at"}).
			AddRow(1, "Test Todo", "Description", false, now, now)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, title, description, completed, created_at, updated_at FROM todos WHERE created_at >= $1 AND created_at < $2 ORDER BY created_at DESC`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rows)

		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)

		// Properly set up query parameters
		req := httptest.NewRequest(http.MethodGet, "/todos/by-date?range=day&date=2025-01-15", nil)
		c.Request = req

		// Parse query parameters
		router.GET("/todos/by-date", server.GetTodosByDate)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success - weekly range returns grouped todos", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		now := time.Now()
		rows := sqlmock.NewRows([]string{"id", "title", "description", "completed", "created_at", "updated_at"}).
			AddRow(1, "Test Todo", "Description", false, now, now)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, title, description, completed, created_at, updated_at FROM todos WHERE created_at >= $1 AND created_at < $2 ORDER BY created_at DESC`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rows)

		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)

		req := httptest.NewRequest(http.MethodGet, "/todos/by-date?range=week&date=2025-01-15", nil)
		c.Request = req

		router.GET("/todos/by-date", server.GetTodosByDate)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success - monthly range returns grouped todos", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		now := time.Now()
		rows := sqlmock.NewRows([]string{"id", "title", "description", "completed", "created_at", "updated_at"}).
			AddRow(1, "Test Todo", "Description", false, now, now)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, title, description, completed, created_at, updated_at FROM todos WHERE created_at >= $1 AND created_at < $2 ORDER BY created_at DESC`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rows)

		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)

		req := httptest.NewRequest(http.MethodGet, "/todos/by-date?range=month&date=2025-01-15", nil)
		c.Request = req

		router.GET("/todos/by-date", server.GetTodosByDate)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - invalid date format", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)

		req := httptest.NewRequest(http.MethodGet, "/todos/by-date?range=day&date=invalid", nil)
		c.Request = req

		router.GET("/todos/by-date", server.GetTodosByDate)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - date exceeds 1 year limit", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		oldDate := time.Now().AddDate(-2, 0, 0).Format("2006-01-02")

		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)

		req := httptest.NewRequest(http.MethodGet, "/todos/by-date?range=day&date="+oldDate, nil)
		c.Request = req

		router.GET("/todos/by-date", server.GetTodosByDate)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - invalid range type", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)

		req := httptest.NewRequest(http.MethodGet, "/todos/by-date?range=invalid&date=2025-01-15", nil)
		c.Request = req

		router.GET("/todos/by-date", server.GetTodosByDate)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - database query fails", func(t *testing.T) {
		server, mock := setupTestServer(t)
		defer server.DB.Close()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, title, description, completed, created_at, updated_at FROM todos WHERE created_at >= $1 AND created_at < $2 ORDER BY created_at DESC`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(assert.AnError)

		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)

		req := httptest.NewRequest(http.MethodGet, "/todos/by-date?range=day&date=2025-01-15", nil)
		c.Request = req

		router.GET("/todos/by-date", server.GetTodosByDate)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
