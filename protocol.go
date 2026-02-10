package main

const (
	// MaxPluginNameLength is the maximum allowed length for plugin names
	MaxPluginNameLength = 64
	// PluginNamePattern is the regex pattern for valid plugin names
	PluginNamePattern = `^[a-zA-Z0-9-]+$`
)

// HandshakeResponse represents the server's response to a handshake
type HandshakeResponse struct {
	Success bool
	Error   string
}
