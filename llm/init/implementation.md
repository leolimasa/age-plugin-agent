# Implementation Plan

## Overview

This implementation creates an age plugin agent in Go that intercepts age plugin calls and forwards them to a remote server via Unix domain sockets.

## Data Structures

### Configuration and State

**File: `config.go`**

```go
type Config struct {
    SocketPath string
}
```

**Purpose**: Store runtime configuration such as Unix socket file path.

### Protocol Messages

**File: `protocol.go`**

```go
const (
    MaxPluginNameLength = 64
    PluginNamePattern   = `^[a-zA-Z0-9-]+$`
)

type HandshakeResponse struct {
    Success bool
    Error   string
}
```

**Purpose**: Define constants and types for the handshake protocol between client and server.

## Functions by Module

### File: `main.go`

#### `main()` [req.wtb211] [req.csnye0]

**Description**: Entry point for the age-plugin-agent binary. Implements binary name detection logic to determine the execution mode:
1. Extract the binary name from `os.Args[0]`
2. Parse the base name (after any directory path)
3. If the name starts with `age-plugin-`, extract the plugin name suffix and automatically execute in proxy mode for that plugin [req.tjn27i]
4. Otherwise, parse and dispatch to the appropriate subcommand handler based on CLI arguments [req.bpo0ci]

#### `getPluginNameFromBinaryName(binaryPath string) (string, bool)` [req.csnye0] [req.tjn27i]

**Description**: Helper function to extract plugin name from binary path. Returns the plugin name and a boolean indicating whether this is a plugin-named binary:
1. Extract base name using `filepath.Base()`
2. Check if it starts with `age-plugin-` prefix
3. If yes, return the suffix as plugin name and true
4. Otherwise, return empty string and false

### File: `cmd/intercept.go` or `intercept.go`

#### `runIntercept(plugins []string, shell string)` [req.nc6prq] [req.02pjoj]

**Description**: Implementation of the `intercept` subcommand. Creates a temporary directory with symlinks and spawns a shell:
1. Parse the comma-separated list of plugin names
2. Create a temporary directory (use `os.MkdirTemp()`)
3. Get the current executable path using `os.Executable()`
4. For each plugin name, create a symlink `age-plugin-<name>` pointing to the current executable
5. Prepend the temporary directory to `$PATH` environment variable
6. Determine which shell to use (parameter if provided, otherwise `$SHELL`, fallback to `/bin/sh`)
7. Spawn the shell with the modified environment using `exec.Command()`
8. Wait for the shell to exit
9. Clean up the temporary directory on exit (using `defer`)

### File: `cmd/proxy.go` or `proxy.go`

#### `runProxy(pluginName string)` [req.nc6prq] [req.erkwws]

**Description**: Implementation of the `proxy` subcommand. Connects to the server and proxies stdin/stdout:
1. Validate the socket path from configuration or environment variable
2. Connect to the Unix domain socket using `net.Dial("unix", socketPath)`
3. Perform the client-side handshake (call `performClientHandshake()`)
4. If handshake fails, write error to stderr and exit with non-zero status
5. If handshake succeeds, start bidirectional proxying:
   - Launch a goroutine to copy from stdin to socket (`io.Copy(conn, os.Stdin)`)
   - In the main goroutine, copy from socket to stdout (`io.Copy(os.Stdout, conn)`)
   - Wait for both copies to complete
6. Close the connection and exit

#### `performClientHandshake(conn net.Conn, pluginName string) error` [req.erkwws]

**Description**: Handles the client side of the handshake protocol:
1. Validate plugin name format using regex `^[a-zA-Z0-9-]+$` and max length 64
2. Write plugin name followed by newline to the connection: `fmt.Fprintf(conn, "%s\n", pluginName)`
3. Read the response line using `bufio.NewReader(conn).ReadString('\n')`
4. Parse the response:
   - If it starts with `OK`, return nil (success)
   - If it starts with `ERROR`, extract the error message and return an error
   - Otherwise, return an error for unexpected response format
5. Set appropriate read timeout to prevent indefinite blocking

### File: `cmd/server.go` or `server.go`

#### `runServer(socketPath string)` [req.nc6prq] [req.lv5ejb] [req.nytk77]

**Description**: Implementation of the `server` subcommand. Starts a Unix domain socket server:
1. Remove any existing socket file at the path (handle `os.ErrNotExist` gracefully)
2. Create a Unix domain socket listener using `net.Listen("unix", socketPath)`
3. Set appropriate permissions on the socket file (e.g., 0600 for security)
4. Set up signal handling (SIGINT, SIGTERM) for graceful shutdown
5. Enter accept loop:
   - Accept incoming connections
   - For each connection, spawn a goroutine to handle it (`handleConnection()`)
6. On shutdown signal, close the listener and clean up the socket file

**Note**: Server emits operational logs to stdout (e.g., server started, connection accepted, plugin executed, errors encountered). This allows operators to monitor server activity and debug issues.

#### `handleConnection(conn net.Conn)` [req.lv5ejb]

**Description**: Handles a single client connection. Called in a goroutine for each accepted connection:
1. Ensure connection is closed when function exits (using `defer conn.Close()`)
2. Perform server-side handshake (call `performServerHandshake()`)
3. If handshake fails, the handshake function handles sending the error response
4. If handshake succeeds, receive the plugin command to execute
5. Start bidirectional proxying (call `proxyToPlugin()`)
6. Wait for the plugin to exit
7. Close the connection (handled by defer)

#### `performServerHandshake(conn net.Conn) (string, error)` [req.lv5ejb]

**Description**: Handles the server side of the handshake protocol:
1. Set a read timeout on the connection (e.g., 10 seconds)
2. Read the plugin name line using `bufio.NewReader(conn).ReadString('\n')`
3. Trim whitespace from the plugin name
4. Validate the plugin name:
   - Check length <= 64 characters
   - Check format using regex `^[a-zA-Z0-9-]+$`
   - If invalid, send `ERROR invalid plugin name: <name>\n` and return error
5. Search for the plugin binary (call `findPluginBinary()`)
6. If plugin not found or not executable, send appropriate error response and return error
7. If validation succeeds, send `OK\n` to the client
8. Return the plugin name for further processing

#### `findPluginBinary(pluginName string) (string, error)`

**Description**: Searches `$PATH` for the age plugin binary:
1. Construct the binary name: `age-plugin-<pluginName>`
2. Use `exec.LookPath()` to search for the binary in `$PATH`
3. If found, verify it's executable:
   - Use `os.Stat()` to get file info
   - Check file mode has execute bit set
4. Return the full path if found and executable
5. Return appropriate error if not found or not executable

#### `proxyToPlugin(conn net.Conn, pluginPath string)` [req.lv5ejb]

**Description**: Spawns the plugin subprocess and proxies data bidirectionally:
1. Create a command using `exec.Command(pluginPath)`
2. Set up stdin/stdout pipes for the plugin process
3. Start the plugin process
4. Launch two goroutines for bidirectional copying:
   - Goroutine 1: Copy from socket to plugin stdin
   - Goroutine 2: Copy from plugin stdout to socket
5. Wait for the plugin process to exit using `cmd.Wait()`
6. Close pipes and connection (errors from plugin are propagated to the age client via stdout)

### File: `util.go` or `validation.go`

#### `validatePluginName(name string) error`

**Description**: Validates plugin name format according to protocol requirements:
1. Check if name is empty (return error if so)
2. Check if length exceeds 64 characters (return error if so)
3. Compile and match against regex `^[a-zA-Z0-9-]+$`
4. Return nil if valid, descriptive error otherwise

Used by both client and server for consistent validation.

### File: `socket.go` or `config.go`

#### `getSocketPath() string` [req.nytk77]

**Description**: Determines the Unix domain socket path to use:
1. Check for environment variable (e.g., `AGE_PLUGIN_AGENT_SOCKET`)
2. If not set, use a default path (e.g., `/tmp/age-plugin-agent.sock` or `~/.age-plugin-agent.sock`)
3. Return the resolved path

This function is used by both client and server to ensure they use the same socket location.

## Implementation Notes

### Functional Programming Approach

- All functions are designed to be as pure as possible
- Side effects (I/O operations, process spawning) are isolated in dedicated functions
- Helper functions like `validatePluginName()` and `getPluginNameFromBinaryName()` are pure functions
- State mutation is limited to necessary I/O operations

### Early Returns

- Validation functions return early on first error
- Handshake functions check and return at each validation step
- Error handling uses early returns to reduce nesting

### Error Handling

- All errors are propagated with descriptive messages
- Server sends structured error responses to clients
- Client displays errors to stderr before exiting
- Network timeouts prevent indefinite blocking

### Logging

- Server emits operational logs to stdout for monitoring and debugging
- Log messages include: server startup, connection events, plugin execution, and errors
- Client proxy mode operates silently (no logs) to avoid interfering with age protocol on stdout

### Security Considerations

- Plugin name validation prevents path traversal attacks
- Symlinks are followed but validated
- Socket permissions restrict access
- No global variables; configuration passed through function parameters or struct fields

### Concurrency

- Server handles each connection in a separate goroutine
- Bidirectional proxying uses goroutines for concurrent reads/writes
- Proper cleanup with defer statements
- No shared mutable state between goroutines

## Requirement Coverage

- [req.wtb211] - Go executable created
- [req.csnye0] - Binary name detection implemented in `main()`
- [req.tjn27i] - Automatic proxy mode when called as `age-plugin-*`
- [req.bpo0ci] - Normal subcommand parsing when called directly
- [req.nc6prq] - Subcommands implemented: `intercept`, `proxy`, `server`
- [req.02pjoj] - Intercept creates symlinks for specified plugins
- [req.erkwws] - Proxy performs handshake and proxies stdin/stdout
- [req.lv5ejb] - Server listens on socket, performs handshake, and proxies to plugin
- [req.nytk77] - Unix domain socket used for communication
