package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// runIntercept implements the intercept subcommand
func runIntercept(plugins []string, shell string) error {
	// Validate plugin names
	for _, plugin := range plugins {
		if err := validatePluginName(plugin); err != nil {
			return fmt.Errorf("invalid plugin name %q: %w", plugin, err)
		}
	}

	// Create temporary directory for symlinks
	tempDir, err := os.MkdirTemp("", "age-plugin-agent-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Get current executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Create symlinks for each plugin
	for _, plugin := range plugins {
		linkName := filepath.Join(tempDir, "age-plugin-"+plugin)
		if err := os.Symlink(exePath, linkName); err != nil {
			return fmt.Errorf("failed to create symlink for %q: %w", plugin, err)
		}
	}

	// Determine which shell to use
	if shell == "" {
		shell = os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
	}

	// Prepare environment with modified PATH
	env := os.Environ()
	pathUpdated := false
	for i, e := range env {
		if len(e) > 5 && e[:5] == "PATH=" {
			env[i] = "PATH=" + tempDir + string(os.PathListSeparator) + e[5:]
			pathUpdated = true
			break
		}
	}
	if !pathUpdated {
		env = append(env, "PATH="+tempDir)
	}

	// Spawn shell with modified environment
	fmt.Printf("Starting shell with intercepted plugins: %v\n", plugins)
	fmt.Printf("Plugin binaries available in: %s\n", tempDir)

	cmd := exec.Command(shell)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("shell exited with error: %w", err)
	}

	return nil
}
