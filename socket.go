package main

import (
	"os"
	"path/filepath"
)

// getSocketPath determines the Unix domain socket path to use
func getSocketPath() string {
	// Check for environment variable first
	if socketPath := os.Getenv("AGE_PLUGIN_AGENT_SOCKET"); socketPath != "" {
		return socketPath
	}

	// Use default path in user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to /tmp if home directory cannot be determined
		return "/tmp/age-plugin-agent.sock"
	}

	return filepath.Join(homeDir, ".age-plugin-agent.sock")
}
