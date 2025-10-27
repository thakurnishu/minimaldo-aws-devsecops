package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
)

func setupDB(cfg *Config) (db *sql.DB) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		slog.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}

	// Initialize database
	if err := initDB(db); err != nil {
		slog.Error("Failed to initialize database", "error",err)
	}

	return db
}

func initDB(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS todos (
		id SERIAL PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		description TEXT,
		completed BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE OR REPLACE FUNCTION update_updated_at_column()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.updated_at = CURRENT_TIMESTAMP;
		RETURN NEW;
	END;
	$$ language 'plpgsql';

	DROP TRIGGER IF EXISTS update_todos_updated_at ON todos;
	CREATE TRIGGER update_todos_updated_at
		BEFORE UPDATE ON todos
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();
	`

	_, err := db.Exec(query)
	return err
}
