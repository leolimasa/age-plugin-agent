package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

// performClientHandshake handles the client side of the handshake protocol
func performClientHandshake(conn net.Conn, pluginName string) error {
	// Validate plugin name
	if err := validatePluginName(pluginName); err != nil {
		return fmt.Errorf("invalid plugin name: %w", err)
	}

	// Set read timeout for handshake
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Send plugin name
	if _, err := fmt.Fprintf(conn, "%s\n", pluginName); err != nil {
		return fmt.Errorf("failed to send plugin name: %w", err)
	}

	// Read response
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read handshake response: %w", err)
	}

	response = strings.TrimSpace(response)

	// Parse response
	if response == "OK" {
		// Clear read deadline for data proxying
		conn.SetReadDeadline(time.Time{})
		return nil
	}

	if strings.HasPrefix(response, "ERROR ") {
		errorMsg := strings.TrimPrefix(response, "ERROR ")
		return fmt.Errorf("server error: %s", errorMsg)
	}

	return fmt.Errorf("unexpected handshake response: %s", response)
}

// runProxy implements the proxy subcommand
func runProxy(pluginName string) error {
	// Get socket path
	socketPath := getSocketPath()

	// Connect to Unix domain socket
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to server at %s: %w", socketPath, err)
	}
	defer conn.Close()

	// Perform handshake
	if err := performClientHandshake(conn, pluginName); err != nil {
		return err
	}

	// Start bidirectional proxying
	done := make(chan error, 2)

	// Goroutine: stdin -> socket
	go func() {
		_, err := io.Copy(conn, os.Stdin)
		done <- err
	}()

	// Main thread: socket -> stdout
	go func() {
		_, err := io.Copy(os.Stdout, conn)
		done <- err
	}()

	// Wait for either direction to complete
	err = <-done

	// Close connection to stop the other goroutine
	conn.Close()

	// Wait for the second goroutine to finish
	<-done

	return err
}
