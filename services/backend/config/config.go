package config

import (
	"log/slog"
	"os"

	"github.com/thakurnishu/MinimalDo/utils"
)

type Config struct {
	// general
	Port        string
	FrontendURL string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBName     string
	DBPassword string

	// otel
	ServiceName                  string
	OtelExporterOtlpEndpointGRPC string

	// Logs
	EnableConsoleLog bool
	LogLevel         slog.Level // values: debug, info, warn, error
}

func LoadConfig() *Config {
	// All configuration values are strictly required using utils.GetEnv.
	// If any of these environment variables are missing or empty,
	// utils.GetEnv will log an error and call os.Exit(1).

	cfg := &Config{
		// General
		Port:        utils.GetEnv("PORT"),
		FrontendURL: utils.GetEnv("FRONTEND_URL"),
		// Database
		DBHost:     utils.GetEnv("DB_HOST"),
		DBPort:     utils.GetEnv("DB_PORT"),
		DBUser:     utils.GetEnv("DB_USER"),
		DBName:     utils.GetEnv("DB_NAME"),
		DBPassword: utils.GetEnv("DB_PASSWORD"),
		// Otel
		ServiceName:                  utils.GetEnv("APP_NAME"),
		OtelExporterOtlpEndpointGRPC: utils.GetEnv("OTEL_EXPORTER_OTLP_ENDPOINT_GRPC"),
		// Logs - EnableConsoleLog must also be strictly present
		EnableConsoleLog: utils.GetEnv("ENABLE_CONSOLE_LOG") == "true",
	}

	logLevelStr := utils.GetEnv("LOG_LEVEL")

	switch logLevelStr {
	case "debug":
		cfg.LogLevel = slog.LevelDebug
	case "info":
		cfg.LogLevel = slog.LevelInfo
	case "warn":
		cfg.LogLevel = slog.LevelWarn
	case "error":
		cfg.LogLevel = slog.LevelError
	default:
		// If LOG_LEVEL is present (guaranteed by utils.GetEnv) but not one of the valid strings,
		// we treat this as a configuration error and exit strictly.
		slog.Error("Invalid LOG_LEVEL provided in environment", "level_set", logLevelStr)
		os.Exit(1)
	}

	return cfg
}
