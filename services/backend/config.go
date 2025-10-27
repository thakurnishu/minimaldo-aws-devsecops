package main

import (
	"log/slog"
)

type Config struct {
	// general
	Port string
	FrontendURL string

	// Database
	DBHost string
	DBPort string
	DBUser string
	DBName string
	DBPassword string
	
	// otel
	ServiceName string
	OtelExporterOtlpEndpointGRPC string

	// Logs
	EnableConsoleLog bool
	LogLevel slog.Level // values: debug, info, warn, error
}

func loadConfig() *Config {
	cfg := &Config{
		// General
		Port: GetEnv("PORT"),
		FrontendURL: GetEnv("FRONTEND_URL"),
		// Database
		DBHost: GetEnv("DB_HOST"),
		DBPort: GetEnv("DB_PORT"),
		DBUser: GetEnv("DB_USER"),
		DBName: GetEnv("DB_NAME"),
		DBPassword: GetEnv("DB_PASSWORD"),
		// Otel
		ServiceName: GetEnv("APP_NAME"),
		OtelExporterOtlpEndpointGRPC: GetEnv("OTEL_EXPORTER_OTLP_ENDPOINT_GRPC"),
		// Logs
		EnableConsoleLog: GetEnv("ENABLE_CONSOLE_LOG") == "true",
	}

	switch GetEnv("LOG_LEVEL") {
	case "debug":
		cfg.LogLevel = slog.LevelDebug
	case "info": 
		cfg.LogLevel = slog.LevelInfo
	case "warn": 
		cfg.LogLevel = slog.LevelWarn
	case "error": 
		cfg.LogLevel = slog.LevelError
	default:
		cfg.LogLevel = slog.LevelInfo
	}

	return cfg
}
