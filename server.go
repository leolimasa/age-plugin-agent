package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// findPluginBinary searches $PATH for the age plugin binary
func findPluginBinary(pluginName string) (string, error) {
	binaryName := "age-plugin-" + pluginName

	// Search for binary in PATH
	path, err := exec.LookPath(binaryName)
	if err != nil {
		return "", fmt.Errorf("plugin not found: %s", pluginName)
	}

	// Verify it's executable
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("plugin not found: %s", pluginName)
	}

	// Check if it's executable
	if info.Mode()&0111 == 0 {
		return "", fmt.Errorf("plugin not executable: %s", path)
	}

	return path, nil
}

// performServerHandshake handles the server side of the handshake protocol
func performServerHandshake(conn net.Conn) (string, error) {
	// Set read timeout for handshake
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return "", fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Read plugin name
	reader := bufio.NewReader(conn)
	pluginNameLine, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read plugin name: %w", err)
	}

	pluginName := strings.TrimSpace(pluginNameLine)

	// Validate plugin name
	if err := validatePluginName(pluginName); err != nil {
		errMsg := fmt.Sprintf("ERROR invalid plugin name: %s\n", pluginName)
		conn.Write([]byte(errMsg))
		return "", fmt.Errorf("invalid plugin name: %s", pluginName)
	}

	// Search for plugin binary
	pluginPath, err := findPluginBinary(pluginName)
	if err != nil {
		var errMsg string
		if strings.Contains(err.Error(), "not found") {
			errMsg = fmt.Sprintf("ERROR plugin not found: %s\n", pluginName)
		} else if strings.Contains(err.Error(), "not executable") {
			errMsg = fmt.Sprintf("ERROR plugin not executable: %s\n", pluginPath)
		} else {
			errMsg = fmt.Sprintf("ERROR %s\n", err.Error())
		}
		conn.Write([]byte(errMsg))
		return "", err
	}

	// Send OK response
	if _, err := conn.Write([]byte("OK\n")); err != nil {
		return "", fmt.Errorf("failed to send OK response: %w", err)
	}

	// Clear read deadline for data proxying
	conn.SetReadDeadline(time.Time{})

	return pluginPath, nil
}

// handleConnection handles a single client connection (stub for now)
func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Perform handshake
	pluginPath, err := performServerHandshake(conn)
	if err != nil {
		// Error already sent to client
		fmt.Fprintf(os.Stderr, "Handshake failed: %v\n", err)
		return
	}

	fmt.Printf("Handshake successful, plugin: %s\n", pluginPath)

	// Proxy to plugin (will be implemented in Phase 6)
	if err := proxyToPlugin(conn, pluginPath); err != nil {
		fmt.Fprintf(os.Stderr, "Plugin proxy error: %v\n", err)
	}
}

// runServer implements the server subcommand
func runServer(socketPath string) error {
	// Remove existing socket file
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	// Create Unix domain socket listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to create socket listener: %w", err)
	}
	defer listener.Close()
	defer os.Remove(socketPath)

	// Set socket permissions
	if err := os.Chmod(socketPath, 0600); err != nil {
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	fmt.Printf("Server started, listening on: %s\n", socketPath)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Channel to signal server to stop
	stopChan := make(chan bool, 1)

	// Goroutine to handle signals
	go func() {
		<-sigChan
		fmt.Println("\nReceived shutdown signal, stopping server...")
		stopChan <- true
		listener.Close()
	}()

	// Accept loop
	for {
		select {
		case <-stopChan:
			fmt.Println("Server stopped")
			return nil
		default:
			// Set a timeout for Accept to allow checking stopChan
			if conn, err := listener.Accept(); err != nil {
				// Check if error is due to listener being closed
				if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
					return nil
				}
				fmt.Fprintf(os.Stderr, "Accept error: %v\n", err)
			} else {
				fmt.Printf("Connection accepted from client\n")
				go handleConnection(conn)
			}
		}
	}
}

// proxyToPlugin spawns the plugin subprocess and proxies data bidirectionally
func proxyToPlugin(conn net.Conn, pluginPath string) error {
	// Create command for the plugin
	cmd := exec.Command(pluginPath)

	// Set up stdin pipe
	pluginStdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Set up stdout pipe
	pluginStdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Connect stderr to our stderr for debugging
	cmd.Stderr = os.Stderr

	// Start the plugin process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start plugin: %w", err)
	}

	fmt.Printf("Plugin started: %s (PID: %d)\n", pluginPath, cmd.Process.Pid)

	// Channel to collect errors from goroutines
	done := make(chan error, 2)

	// Goroutine 1: socket -> plugin stdin
	go func() {
		_, err := io.Copy(pluginStdin, conn)
		pluginStdin.Close()
		done <- err
	}()

	// Goroutine 2: plugin stdout -> socket
	go func() {
		_, err := io.Copy(conn, pluginStdout)
		done <- err
	}()

	// Wait for plugin process to exit
	processErr := cmd.Wait()

	// Close connection to stop goroutines
	conn.Close()

	// Collect errors from goroutines
	err1 := <-done
	err2 := <-done

	fmt.Printf("Plugin exited: %s (PID: %d)\n", pluginPath, cmd.Process.Pid)

	// Return first non-nil error
	if processErr != nil {
		return fmt.Errorf("plugin process error: %w", processErr)
	}
	if err1 != nil {
		return fmt.Errorf("socket to plugin error: %w", err1)
	}
	if err2 != nil {
		return fmt.Errorf("plugin to socket error: %w", err2)
	}

	return nil
}
