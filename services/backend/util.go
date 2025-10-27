package main

import (
	"log/slog"
	"os"
)

func GetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		slog.Error("Environment missing", "key", key)
		os.Exit(1)
	}
	return value
}
