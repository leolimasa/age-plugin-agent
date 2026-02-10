package main

import "testing"

func TestGetPluginNameFromBinaryName(t *testing.T) {
	tests := []struct {
		name           string
		binaryPath     string
		wantPluginName string
		wantIsPlugin   bool
	}{
		{
			name:           "symlink with plugin name",
			binaryPath:     "age-plugin-yubikey",
			wantPluginName: "yubikey",
			wantIsPlugin:   true,
		},
		{
			name:           "symlink with full path",
			binaryPath:     "/tmp/test/age-plugin-test",
			wantPluginName: "test",
			wantIsPlugin:   true,
		},
		{
			name:           "symlink with hyphenated name",
			binaryPath:     "age-plugin-my-plugin",
			wantPluginName: "my-plugin",
			wantIsPlugin:   true,
		},
		{
			name:           "direct binary invocation",
			binaryPath:     "age-plugin-agent",
			wantPluginName: "",
			wantIsPlugin:   false,
		},
		{
			name:           "direct binary with path",
			binaryPath:     "/usr/local/bin/age-plugin-agent",
			wantPluginName: "",
			wantIsPlugin:   false,
		},
		{
			name:           "other binary",
			binaryPath:     "some-other-binary",
			wantPluginName: "",
			wantIsPlugin:   false,
		},
		{
			name:           "relative path to plugin",
			binaryPath:     "./age-plugin-custom",
			wantPluginName: "custom",
			wantIsPlugin:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPluginName, gotIsPlugin := getPluginNameFromBinaryName(tt.binaryPath)
			if gotPluginName != tt.wantPluginName {
				t.Errorf("getPluginNameFromBinaryName(%q) pluginName = %q, want %q",
					tt.binaryPath, gotPluginName, tt.wantPluginName)
			}
			if gotIsPlugin != tt.wantIsPlugin {
				t.Errorf("getPluginNameFromBinaryName(%q) isPlugin = %v, want %v",
					tt.binaryPath, gotIsPlugin, tt.wantIsPlugin)
			}
		})
	}
}
