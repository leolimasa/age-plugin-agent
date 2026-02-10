package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetSocketPath(t *testing.T) {
	// Save original environment variable
	originalEnv := os.Getenv("AGE_PLUGIN_AGENT_SOCKET")
	defer func() {
		if originalEnv != "" {
			os.Setenv("AGE_PLUGIN_AGENT_SOCKET", originalEnv)
		} else {
			os.Unsetenv("AGE_PLUGIN_AGENT_SOCKET")
		}
	}()

	t.Run("with environment variable set", func(t *testing.T) {
		expectedPath := "/custom/path/to/socket.sock"
		os.Setenv("AGE_PLUGIN_AGENT_SOCKET", expectedPath)

		result := getSocketPath()
		if result != expectedPath {
			t.Errorf("getSocketPath() = %q, want %q", result, expectedPath)
		}
	})

	t.Run("without environment variable", func(t *testing.T) {
		os.Unsetenv("AGE_PLUGIN_AGENT_SOCKET")

		result := getSocketPath()

		// Should return either home directory path or /tmp fallback
		homeDir, err := os.UserHomeDir()
		if err != nil {
			// Should use /tmp fallback
			expectedPath := "/tmp/age-plugin-agent.sock"
			if result != expectedPath {
				t.Errorf("getSocketPath() = %q, want %q", result, expectedPath)
			}
		} else {
			// Should use home directory
			expectedPath := filepath.Join(homeDir, ".age-plugin-agent.sock")
			if result != expectedPath {
				t.Errorf("getSocketPath() = %q, want %q", result, expectedPath)
			}
		}
	})
}
