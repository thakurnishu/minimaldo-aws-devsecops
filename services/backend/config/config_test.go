package config_test

import (
	"log/slog"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thakurnishu/MinimalDo/config"
)

// Helper function to set up default environment variables required by LoadConfig.
func setupTestEnv() {
	os.Setenv("PORT", "8080")
	os.Setenv("FRONTEND_URL", "http://localhost:3000")
	os.Setenv("DB_HOST", "db-host")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "user")
	os.Setenv("DB_NAME", "db-name")
	os.Setenv("DB_PASSWORD", "secret")
	os.Setenv("APP_NAME", "MinimalDoService")
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT_GRPC", "otel.endpoint:4317")
	os.Setenv("ENABLE_CONSOLE_LOG", "false")
	os.Setenv("LOG_LEVEL", "info") // Set a default valid level
}

// Helper function to clean up all set environment variables.
func cleanupTestEnv() {
	os.Unsetenv("PORT")
	os.Unsetenv("FRONTEND_URL")
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_PORT")
	os.Unsetenv("DB_USER")
	os.Unsetenv("DB_NAME")
	os.Unsetenv("DB_PASSWORD")
	os.Unsetenv("APP_NAME")
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT_GRPC")
	os.Unsetenv("ENABLE_CONSOLE_LOG")
	os.Unsetenv("LOG_LEVEL")
}

func TestLoadConfig_Success(t *testing.T) {
	// Setup and Teardown
	setupTestEnv()
	defer cleanupTestEnv()

	cfg := config.LoadConfig()

	// Assert that all values were loaded correctly
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "http://localhost:3000", cfg.FrontendURL)
	assert.Equal(t, "db-host", cfg.DBHost)
	assert.Equal(t, "5432", cfg.DBPort)
	assert.Equal(t, "user", cfg.DBUser)
	assert.Equal(t, "db-name", cfg.DBName)
	assert.Equal(t, "secret", cfg.DBPassword)
	assert.Equal(t, "MinimalDoService", cfg.ServiceName)
	assert.Equal(t, "otel.endpoint:4317", cfg.OtelExporterOtlpEndpointGRPC)
	assert.False(t, cfg.EnableConsoleLog)
	assert.Equal(t, slog.LevelInfo, cfg.LogLevel)
}

func TestLoadConfig_LogLevels(t *testing.T) {
	// Setup and Teardown
	setupTestEnv()
	defer cleanupTestEnv()

	tests := []struct {
		name     string
		envValue string
		expected slog.Level
	}{
		{"debug level", "debug", slog.LevelDebug},
		{"info level", "info", slog.LevelInfo},
		{"warn level", "warn", slog.LevelWarn},
		{"error level", "error", slog.LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("LOG_LEVEL", tt.envValue)

			cfg := config.LoadConfig()
			assert.Equal(t, tt.expected, cfg.LogLevel)
		})
	}
}

func TestLoadConfig_EnableConsoleLog(t *testing.T) {
	// Setup and Teardown
	setupTestEnv()
	defer cleanupTestEnv()

	t.Run("Enabled is true", func(t *testing.T) {
		os.Setenv("ENABLE_CONSOLE_LOG", "true")
		cfg := config.LoadConfig()
		assert.True(t, cfg.EnableConsoleLog)
	})

	t.Run("Enabled is false (not 'true')", func(t *testing.T) {
		os.Setenv("ENABLE_CONSOLE_LOG", "0") // Must still be set to avoid utils.GetEnv exit
		cfg := config.LoadConfig()
		assert.False(t, cfg.EnableConsoleLog)
	})
}

// TestLoadConfig_StrictFailures uses the subprocess technique to safely test the os.Exit(1) call
// for missing keys and invalid values.
func TestLoadConfig_StrictFailures(t *testing.T) {
	const exitFlag = "TEST_SHOULD_EXIT_1"

	// If the subprocess flag is set, this is the subprocess running the test logic.
	if os.Getenv(exitFlag) == "1" {
		// This block executes the logic that is expected to fail.

		// Determine the test case from the environment
		testCase := os.Getenv("TEST_CASE")
		keyToUnset := os.Getenv("UNSET_KEY")
		invalidValue := os.Getenv("INVALID_LOG_LEVEL")

		// 1. Setup all variables
		setupTestEnv()

		if testCase == "missing_key" {
			// 2a. Unset the target key, which will fail inside utils.GetEnv
			os.Unsetenv(keyToUnset)
		} else if testCase == "invalid_log_level" {
			// 2b. Set an invalid log level, which will fail inside LoadConfig's switch statement
			os.Setenv("LOG_LEVEL", invalidValue)
		}

		// 3. Run LoadConfig, which should cause os.Exit(1)
		config.LoadConfig()
		return // Should not be reached
	}

	// Main process logic: Define and execute tests in a separate command.
	tests := []struct {
		name          string
		testCase      string
		unsetKey      string // Used if testCase is "missing_key"
		invalidLevel  string // Used if testCase is "invalid_log_level"
		expectedError string // Part of the expected log message
	}{
		{
			name:          "Failure - Missing PORT key",
			testCase:      "missing_key",
			unsetKey:      "PORT",
			expectedError: "key=PORT",
		},
		{
			name:          "Failure - Missing DB_HOST key",
			testCase:      "missing_key",
			unsetKey:      "DB_HOST",
			expectedError: "key=DB_HOST",
		},
		{
			name:          "Failure - Invalid LOG_LEVEL value",
			testCase:      "invalid_log_level",
			invalidLevel:  "verbose",
			expectedError: "Invalid LOG_LEVEL provided in environment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestLoadConfig_StrictFailures$")

			// Pass environment variables to the subprocess
			cmd.Env = append(os.Environ(),
				exitFlag+"=1",
				"TEST_CASE="+tt.testCase,
				"UNSET_KEY="+tt.unsetKey,
				"INVALID_LOG_LEVEL="+tt.invalidLevel,
			)

			// Capture CombinedOutput
			output, err := cmd.CombinedOutput()

			// Assertions for failure
			if e, ok := err.(*exec.ExitError); ok {
				assert.False(t, e.Success(), "Expected command to exit with non-zero status")
				assert.Equal(t, 1, e.ExitCode(), "Expected exit code 1")

				outputStr := string(output)
				assert.Contains(t, outputStr, "ERROR", "Expected ERROR log level in output")
				assert.Contains(t, outputStr, tt.expectedError, "Expected specific error message in output")
			} else {
				t.Fatalf("Command finished unexpectedly: %v", err)
			}
		})
	}
}
