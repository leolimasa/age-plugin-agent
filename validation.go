package main

import (
	"fmt"
	"regexp"
)

var pluginNameRegex = regexp.MustCompile(PluginNamePattern)

// validatePluginName validates plugin name format according to protocol requirements
func validatePluginName(name string) error {
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	if len(name) > MaxPluginNameLength {
		return fmt.Errorf("plugin name exceeds maximum length of %d characters", MaxPluginNameLength)
	}
	if !pluginNameRegex.MatchString(name) {
		return fmt.Errorf("plugin name contains invalid characters (only alphanumeric and hyphens allowed)")
	}
	return nil
}
