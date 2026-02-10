# TODO: age-plugin-agent Implementation

## Phase 1: Project Setup and Core Data Structures ✓

- [x] Initialize Go module with `go mod init`
- [x] Create project directory structure
- [x] Create `config.go` with Config struct [req.nytk77]
- [x] Create `protocol.go` with protocol constants and types
  - [x] Define `MaxPluginNameLength = 64`
  - [x] Define `PluginNamePattern = ^[a-zA-Z0-9-]+$`
  - [x] Define `HandshakeResponse` struct
- [x] Create `validation.go` with `validatePluginName()` function
- [x] Create `socket.go` with `getSocketPath()` function [req.nytk77]
- [x] **Test Phase 1**: Run unit tests for validation and utility functions
  - [x] Test `validatePluginName()` with valid names (alphanumeric, hyphens)
  - [x] Test `validatePluginName()` with invalid names (special chars, too long, empty)
  - [x] Test `getSocketPath()` with and without environment variable

## Phase 2: Main Entry Point and Binary Name Detection ✓

- [x] Create `main.go` with entry point [req.wtb211] [req.csnye0]
- [x] Implement `getPluginNameFromBinaryName()` helper function [req.csnye0] [req.tjn27i]
  - [x] Extract base name from binary path
  - [x] Check for `age-plugin-` prefix
  - [x] Return plugin name suffix if found
- [x] Implement `main()` function with binary name detection [req.csnye0]
  - [x] Check `os.Args[0]` for binary name
  - [x] If starts with `age-plugin-`, extract plugin name and run proxy mode [req.tjn27i]
  - [x] Otherwise, parse subcommands normally [req.bpo0ci]
- [x] Add CLI argument parsing for subcommands [req.nc6prq]
  - [x] Parse `intercept` subcommand
  - [x] Parse `proxy` subcommand
  - [x] Parse `server` subcommand
- [x] **Test Phase 2**: Build and test binary name detection
  - [x] Build the binary: `go build -o age-plugin-agent`
  - [x] Test direct invocation with `--help` flag
  - [x] Create symlink: `ln -s age-plugin-agent age-plugin-test`
  - [x] Test symlink invocation to verify automatic proxy mode detection

## Phase 3: Intercept Subcommand ✓

- [x] Create `intercept.go` with intercept functionality [req.nc6prq] [req.02pjoj]
- [x] Implement `runIntercept(plugins []string, shell string)` [req.02pjoj]
  - [x] Parse comma-separated plugin names
  - [x] Create temporary directory with `os.MkdirTemp()`
  - [x] Get current executable path with `os.Executable()`
  - [x] Create symlinks for each plugin: `age-plugin-<name>` → current executable
  - [x] Prepend temp directory to `$PATH`
  - [x] Determine shell to use (parameter, `$SHELL`, or `/bin/sh`)
  - [x] Spawn shell with modified environment
  - [x] Clean up temp directory on exit (with defer)
- [x] **Test Phase 3**: Test intercept subcommand
  - [x] Run: `./age-plugin-agent intercept test-plugin`
  - [x] Verify symlinks are created in temp directory
  - [x] Verify `$PATH` includes temp directory
  - [x] Run `which age-plugin-test-plugin` to confirm it's found
  - [x] Exit shell and verify cleanup

## Phase 4: Proxy Subcommand - Client Side ✓

- [x] Create `proxy.go` with proxy functionality [req.nc6prq] [req.erkwws]
- [x] Implement `performClientHandshake(conn net.Conn, pluginName string) error` [req.erkwws]
  - [x] Validate plugin name format (regex and max length)
  - [x] Write plugin name + newline to connection
  - [x] Set read timeout
  - [x] Read response line from connection
  - [x] Parse response: `OK` or `ERROR <message>`
  - [x] Return nil on success, error on failure
- [x] Implement `runProxy(pluginName string)` [req.erkwws]
  - [x] Get socket path from config/environment
  - [x] Connect to Unix domain socket
  - [x] Perform client handshake
  - [x] Exit with error if handshake fails
  - [x] Start bidirectional proxying:
    - [x] Goroutine: stdin → socket
    - [x] Main thread: socket → stdout
  - [x] Close connection on exit
- [x] **Test Phase 4**: Test proxy client (requires mock server)
  - [x] Create mock Unix socket server for testing
  - [x] Test successful handshake with valid plugin name
  - [x] Test handshake failure with invalid plugin name
  - [x] Test handshake failure with non-existent plugin
  - [x] Test bidirectional data flow (echo test)

## Phase 5: Server Subcommand - Socket Setup and Connection Handling ✓

- [x] Create `server.go` with server functionality [req.nc6prq] [req.lv5ejb] [req.nytk77]
- [x] Implement `findPluginBinary(pluginName string) (string, error)`
  - [x] Construct binary name: `age-plugin-<pluginName>`
  - [x] Use `exec.LookPath()` to search `$PATH`
  - [x] Verify file is executable
  - [x] Return full path or error
- [x] Implement `performServerHandshake(conn net.Conn) (string, error)` [req.lv5ejb]
  - [x] Set read timeout on connection
  - [x] Read plugin name line from connection
  - [x] Trim whitespace
  - [x] Validate plugin name (length and format)
  - [x] Send `ERROR invalid plugin name: <name>\n` if invalid
  - [x] Search for plugin binary with `findPluginBinary()`
  - [x] Send appropriate error if plugin not found/not executable
  - [x] Send `OK\n` if validation succeeds
  - [x] Return plugin name for further processing
- [x] Implement `runServer(socketPath string)` [req.lv5ejb] [req.nytk77]
  - [x] Remove existing socket file (ignore `os.ErrNotExist`)
  - [x] Create Unix domain socket listener
  - [x] Set socket file permissions to 0600
  - [x] Set up signal handling (SIGINT, SIGTERM)
  - [x] Enter accept loop
  - [x] Spawn goroutine for each connection: `handleConnection()`
  - [x] Add logging: server started, connections accepted, errors
  - [x] Clean up socket file on shutdown
- [x] **Test Phase 5**: Test server socket setup
  - [x] Start server: `./age-plugin-agent server`
  - [x] Verify socket file is created at expected path
  - [x] Verify socket file permissions (0600)
  - [x] Test server accepts connections with `nc -U <socket>`
  - [x] Test graceful shutdown with SIGINT

## Phase 6: Server Subcommand - Plugin Proxying ✓

- [x] Implement `proxyToPlugin(conn net.Conn, pluginPath string)` [req.lv5ejb]
  - [x] Create command with `exec.Command(pluginPath)`
  - [x] Set up stdin/stdout pipes for plugin process
  - [x] Start plugin process
  - [x] Launch goroutines for bidirectional copying:
    - [x] Goroutine 1: socket → plugin stdin
    - [x] Goroutine 2: plugin stdout → socket
  - [x] Wait for plugin process to exit
  - [x] Close pipes and connection
- [x] Implement `handleConnection(conn net.Conn)` [req.lv5ejb]
  - [x] Add `defer conn.Close()`
  - [x] Perform server-side handshake
  - [x] Exit if handshake fails (error already sent)
  - [x] Receive plugin command to execute
  - [x] Start bidirectional proxying to plugin
  - [x] Wait for plugin exit
- [x] **Test Phase 6**: Test server plugin proxying
  - [x] Create mock age plugin for testing (simple echo script)
  - [x] Start server with mock plugin in `$PATH`
  - [x] Connect with proxy client and verify data flow
  - [x] Test with actual `age` binary if available
  - [x] Test error cases: plugin not found, plugin crashes

## Phase 7: Integration Testing ✓

- [x] **Test end-to-end workflow**:
  - [x] Start server: `./age-plugin-agent server`
  - [x] In new terminal, run intercept: `./age-plugin-agent intercept yubikey` (or test plugin)
  - [x] Verify symlinks are created and in `$PATH`
  - [x] Test age encryption/decryption with intercepted plugin
  - [x] Verify data flows: age → proxy → server → plugin → server → proxy → age
- [x] **Test error handling**:
  - [x] Test with non-existent plugin name
  - [x] Test with invalid plugin name (special characters)
  - [x] Test with plugin binary that's not executable
  - [x] Test with server not running
  - [x] Test connection timeout scenarios
- [x] **Test concurrency**:
  - [x] Run multiple proxy clients simultaneously
  - [x] Verify server handles concurrent connections
  - [x] Test race conditions with rapid connect/disconnect

## Phase 8: Documentation and Polish

- [ ] Add usage examples to README
  - [ ] Document `intercept` subcommand usage
  - [ ] Document `proxy` subcommand usage
  - [ ] Document `server` subcommand usage
  - [ ] Document socket path configuration
  - [ ] Add SSH forwarding example
- [ ] Add logging configuration options
- [ ] Add version information to binary
- [ ] Optimize error messages for clarity
- [ ] **Test Phase 8**: Verify documentation accuracy
  - [ ] Follow README examples step-by-step
  - [ ] Test all documented use cases
  - [ ] Verify SSH forwarding example works

## Requirement Coverage Checklist

- [x] [req.wtb211] - Go executable called `age-plugin-agent` (Phase 2)
- [x] [req.csnye0] - Binary name detection via `os.Args[0]` (Phase 2)
- [x] [req.tjn27i] - Extract plugin name and run in proxy mode when called as `age-plugin-*` (Phase 2)
- [x] [req.bpo0ci] - Parse subcommands normally when called directly (Phase 2)
- [x] [req.nc6prq] - Subcommands: intercept, proxy, server (Phases 3, 4, 5)
- [x] [req.02pjoj] - Intercept creates symlinks for specified plugins (Phase 3)
- [x] [req.erkwws] - Proxy connects to server, performs handshake, proxies stdin/stdout (Phase 4)
- [x] [req.lv5ejb] - Server listens on socket, performs handshake, proxies to plugin (Phases 5, 6)
- [x] [req.nytk77] - Unix domain socket for communication (Phases 1, 4, 5)

## Notes

- Each phase should be completed and tested before moving to the next
- Unit tests should be written alongside implementation code
- Integration tests in Phase 7 verify all components work together
- Socket path can be configured via `AGE_PLUGIN_AGENT_SOCKET` environment variable
