package types

import (
	"time"
)

type Todo struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type GroupedTodos struct {
	Date  string `json:"date"`
	Todos []Todo `json:"todos"`
}
