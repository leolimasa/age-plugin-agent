package main

import (
	"strings"
	"testing"
)

func TestValidatePluginName(t *testing.T) {
	tests := []struct {
		name      string
		pluginName string
		wantErr   bool
	}{
		{
			name:       "valid simple name",
			pluginName: "yubikey",
			wantErr:    false,
		},
		{
			name:       "valid name with hyphens",
			pluginName: "my-plugin-name",
			wantErr:    false,
		},
		{
			name:       "valid name with numbers",
			pluginName: "plugin123",
			wantErr:    false,
		},
		{
			name:       "valid mixed alphanumeric with hyphens",
			pluginName: "test-plugin-v2",
			wantErr:    false,
		},
		{
			name:       "empty name",
			pluginName: "",
			wantErr:    true,
		},
		{
			name:       "name too long",
			pluginName: strings.Repeat("a", MaxPluginNameLength+1),
			wantErr:    true,
		},
		{
			name:       "name with slash",
			pluginName: "plugin/test",
			wantErr:    true,
		},
		{
			name:       "name with dot",
			pluginName: "plugin.test",
			wantErr:    true,
		},
		{
			name:       "path traversal attempt",
			pluginName: "../../../etc/passwd",
			wantErr:    true,
		},
		{
			name:       "name with underscore",
			pluginName: "plugin_test",
			wantErr:    true,
		},
		{
			name:       "name with space",
			pluginName: "plugin test",
			wantErr:    true,
		},
		{
			name:       "max length name",
			pluginName: strings.Repeat("a", MaxPluginNameLength),
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePluginName(tt.pluginName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePluginName(%q) error = %v, wantErr %v", tt.pluginName, err, tt.wantErr)
			}
		})
	}
}
