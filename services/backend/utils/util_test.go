package utils_test

import (
	"bytes"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thakurnishu/MinimalDo/utils"
)

// Define unique keys for testing.
const (
	testEnvKey = "UNIT_TEST_ENV_KEY"
	testEnvVal = "successful_retrieval"
	exitFlag   = "TEST_SHOULD_EXIT_1"
)

// TestGetEnv contains all subtests for the GetEnv utility function.
func TestGetEnv(t *testing.T) {

	// --- Subtest 1: Success Case ---
	t.Run("Success - returns correct value when key is set", func(t *testing.T) {
		// Setup: Set environment variable
		os.Setenv(testEnvKey, testEnvVal)
		// Teardown: Unset environment variable after test
		defer os.Unsetenv(testEnvKey)

		// Execution & Assertion
		val := utils.GetEnv(testEnvKey)
		assert.Equal(t, testEnvVal, val)
	})

	// --- Subtest 2: Failure Case (tests os.Exit(1) behavior) ---
	// This test executes the failure logic in a separate subprocess, allowing the 
	// main test runner to safely check the exit code.
	t.Run("Failure - exits with code 1 when key is missing", func(t *testing.T) {
		// Ensure the environment variable is not set for this test.
		os.Unsetenv(testEnvKey)

		// If the subprocess flag is set, this is the subprocess running the test logic.
		if os.Getenv(exitFlag) == "1" {
			// Subprocess logic: Execute the function that should call os.Exit(1)
			utils.GetEnv(testEnvKey)
			return // Should not be reached
		}

		// Main process logic: Execute the test in a separate command.
		// We use the full test path for the `-test.run` flag to isolate this subtest.
		cmd := exec.Command(os.Args[0], "-test.run=^TestGetEnv/Failure_-_exits_with_code_1_when_key_is_missing$")

		// Pass the exit flag to the subprocess's environment
		cmd.Env = append(os.Environ(), exitFlag+"=1")

		// Capture Stderr to inspect the slog output, which is logged before exiting.
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()

		// Assertion: We expect the command to fail with a non-zero exit code.
		if e, ok := err.(*exec.ExitError); ok {
			assert.False(t, e.Success(), "Expected command to exit with non-zero status")
			assert.Equal(t, 1, e.ExitCode(), "Expected exit code 1")

			// Check that the error message was logged.
			// Changed "level=ERROR" to "ERROR" to match slog's default TextHandler output.
			assert.Contains(t, stderr.String(), "ERROR", "Expected ERROR log level in output")
			assert.Contains(t, stderr.String(), "Environment missing", "Expected log message about missing environment key")
		} else {
			t.Fatalf("Command finished unexpectedly: %v", err)
		}
	})
}
