package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPerformClientHandshake(t *testing.T) {
	tests := []struct {
		name           string
		pluginName     string
		serverResponse string
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful handshake",
			pluginName:     "yubikey",
			serverResponse: "OK\n",
			wantErr:        false,
		},
		{
			name:           "server error - plugin not found",
			pluginName:     "nonexistent",
			serverResponse: "ERROR plugin not found: nonexistent\n",
			wantErr:        true,
			errContains:    "plugin not found",
		},
		{
			name:           "server error - invalid plugin name",
			pluginName:     "test",
			serverResponse: "ERROR invalid plugin name: ../passwd\n",
			wantErr:        true,
			errContains:    "invalid plugin name",
		},
		{
			name:           "invalid plugin name client-side",
			pluginName:     "../../../etc/passwd",
			serverResponse: "",
			wantErr:        true,
			errContains:    "invalid plugin name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pluginName == "../../../etc/passwd" {
				// Client-side validation should fail before connection
				err := performClientHandshake(nil, tt.pluginName)
				if (err != nil) != tt.wantErr {
					t.Errorf("performClientHandshake() error = %v, wantErr %v", err, tt.wantErr)
				}
				if err != nil && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			// Create a mock server for other tests
			socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("test-socket-%d.sock", time.Now().UnixNano()))
			defer os.Remove(socketPath)

			listener, err := net.Listen("unix", socketPath)
			if err != nil {
				t.Fatalf("Failed to create test listener: %v", err)
			}
			defer listener.Close()

			// Start mock server in goroutine
			done := make(chan bool)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				defer conn.Close()

				// Read plugin name
				reader := bufio.NewReader(conn)
				_, err = reader.ReadString('\n')
				if err != nil {
					return
				}

				// Send response
				conn.Write([]byte(tt.serverResponse))
				done <- true
			}()

			// Connect client
			conn, err := net.Dial("unix", socketPath)
			if err != nil {
				t.Fatalf("Failed to connect to test server: %v", err)
			}
			defer conn.Close()

			// Perform handshake
			err = performClientHandshake(conn, tt.pluginName)

			// Wait for server to finish
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("Test timeout")
			}

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("performClientHandshake() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
			}
		})
	}
}
