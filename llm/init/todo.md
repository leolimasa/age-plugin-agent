# TODO: age-plugin-agent Implementation

## Phase 1: Project Setup and Core Data Structures

- [ ] Initialize Go module with `go mod init`
- [ ] Create project directory structure
- [ ] Create `config.go` with Config struct [req.nytk77]
- [ ] Create `protocol.go` with protocol constants and types
  - [ ] Define `MaxPluginNameLength = 64`
  - [ ] Define `PluginNamePattern = ^[a-zA-Z0-9-]+$`
  - [ ] Define `HandshakeResponse` struct
- [ ] Create `validation.go` with `validatePluginName()` function
- [ ] Create `socket.go` with `getSocketPath()` function [req.nytk77]
- [ ] **Test Phase 1**: Run unit tests for validation and utility functions
  - [ ] Test `validatePluginName()` with valid names (alphanumeric, hyphens)
  - [ ] Test `validatePluginName()` with invalid names (special chars, too long, empty)
  - [ ] Test `getSocketPath()` with and without environment variable

## Phase 2: Main Entry Point and Binary Name Detection

- [ ] Create `main.go` with entry point [req.wtb211] [req.csnye0]
- [ ] Implement `getPluginNameFromBinaryName()` helper function [req.csnye0] [req.tjn27i]
  - [ ] Extract base name from binary path
  - [ ] Check for `age-plugin-` prefix
  - [ ] Return plugin name suffix if found
- [ ] Implement `main()` function with binary name detection [req.csnye0]
  - [ ] Check `os.Args[0]` for binary name
  - [ ] If starts with `age-plugin-`, extract plugin name and run proxy mode [req.tjn27i]
  - [ ] Otherwise, parse subcommands normally [req.bpo0ci]
- [ ] Add CLI argument parsing for subcommands [req.nc6prq]
  - [ ] Parse `intercept` subcommand
  - [ ] Parse `proxy` subcommand
  - [ ] Parse `server` subcommand
- [ ] **Test Phase 2**: Build and test binary name detection
  - [ ] Build the binary: `go build -o age-plugin-agent`
  - [ ] Test direct invocation with `--help` flag
  - [ ] Create symlink: `ln -s age-plugin-agent age-plugin-test`
  - [ ] Test symlink invocation to verify automatic proxy mode detection

## Phase 3: Intercept Subcommand

- [ ] Create `intercept.go` with intercept functionality [req.nc6prq] [req.02pjoj]
- [ ] Implement `runIntercept(plugins []string, shell string)` [req.02pjoj]
  - [ ] Parse comma-separated plugin names
  - [ ] Create temporary directory with `os.MkdirTemp()`
  - [ ] Get current executable path with `os.Executable()`
  - [ ] Create symlinks for each plugin: `age-plugin-<name>` → current executable
  - [ ] Prepend temp directory to `$PATH`
  - [ ] Determine shell to use (parameter, `$SHELL`, or `/bin/sh`)
  - [ ] Spawn shell with modified environment
  - [ ] Clean up temp directory on exit (with defer)
- [ ] **Test Phase 3**: Test intercept subcommand
  - [ ] Run: `./age-plugin-agent intercept test-plugin`
  - [ ] Verify symlinks are created in temp directory
  - [ ] Verify `$PATH` includes temp directory
  - [ ] Run `which age-plugin-test-plugin` to confirm it's found
  - [ ] Exit shell and verify cleanup

## Phase 4: Proxy Subcommand - Client Side

- [ ] Create `proxy.go` with proxy functionality [req.nc6prq] [req.erkwws]
- [ ] Implement `performClientHandshake(conn net.Conn, pluginName string) error` [req.erkwws]
  - [ ] Validate plugin name format (regex and max length)
  - [ ] Write plugin name + newline to connection
  - [ ] Set read timeout
  - [ ] Read response line from connection
  - [ ] Parse response: `OK` or `ERROR <message>`
  - [ ] Return nil on success, error on failure
- [ ] Implement `runProxy(pluginName string)` [req.erkwws]
  - [ ] Get socket path from config/environment
  - [ ] Connect to Unix domain socket
  - [ ] Perform client handshake
  - [ ] Exit with error if handshake fails
  - [ ] Start bidirectional proxying:
    - [ ] Goroutine: stdin → socket
    - [ ] Main thread: socket → stdout
  - [ ] Close connection on exit
- [ ] **Test Phase 4**: Test proxy client (requires mock server)
  - [ ] Create mock Unix socket server for testing
  - [ ] Test successful handshake with valid plugin name
  - [ ] Test handshake failure with invalid plugin name
  - [ ] Test handshake failure with non-existent plugin
  - [ ] Test bidirectional data flow (echo test)

## Phase 5: Server Subcommand - Socket Setup and Connection Handling

- [ ] Create `server.go` with server functionality [req.nc6prq] [req.lv5ejb] [req.nytk77]
- [ ] Implement `findPluginBinary(pluginName string) (string, error)`
  - [ ] Construct binary name: `age-plugin-<pluginName>`
  - [ ] Use `exec.LookPath()` to search `$PATH`
  - [ ] Verify file is executable
  - [ ] Return full path or error
- [ ] Implement `performServerHandshake(conn net.Conn) (string, error)` [req.lv5ejb]
  - [ ] Set read timeout on connection
  - [ ] Read plugin name line from connection
  - [ ] Trim whitespace
  - [ ] Validate plugin name (length and format)
  - [ ] Send `ERROR invalid plugin name: <name>\n` if invalid
  - [ ] Search for plugin binary with `findPluginBinary()`
  - [ ] Send appropriate error if plugin not found/not executable
  - [ ] Send `OK\n` if validation succeeds
  - [ ] Return plugin name for further processing
- [ ] Implement `runServer(socketPath string)` [req.lv5ejb] [req.nytk77]
  - [ ] Remove existing socket file (ignore `os.ErrNotExist`)
  - [ ] Create Unix domain socket listener
  - [ ] Set socket file permissions to 0600
  - [ ] Set up signal handling (SIGINT, SIGTERM)
  - [ ] Enter accept loop
  - [ ] Spawn goroutine for each connection: `handleConnection()`
  - [ ] Add logging: server started, connections accepted, errors
  - [ ] Clean up socket file on shutdown
- [ ] **Test Phase 5**: Test server socket setup
  - [ ] Start server: `./age-plugin-agent server`
  - [ ] Verify socket file is created at expected path
  - [ ] Verify socket file permissions (0600)
  - [ ] Test server accepts connections with `nc -U <socket>`
  - [ ] Test graceful shutdown with SIGINT

## Phase 6: Server Subcommand - Plugin Proxying

- [ ] Implement `proxyToPlugin(conn net.Conn, pluginPath string)` [req.lv5ejb]
  - [ ] Create command with `exec.Command(pluginPath)`
  - [ ] Set up stdin/stdout pipes for plugin process
  - [ ] Start plugin process
  - [ ] Launch goroutines for bidirectional copying:
    - [ ] Goroutine 1: socket → plugin stdin
    - [ ] Goroutine 2: plugin stdout → socket
  - [ ] Wait for plugin process to exit
  - [ ] Close pipes and connection
- [ ] Implement `handleConnection(conn net.Conn)` [req.lv5ejb]
  - [ ] Add `defer conn.Close()`
  - [ ] Perform server-side handshake
  - [ ] Exit if handshake fails (error already sent)
  - [ ] Receive plugin command to execute
  - [ ] Start bidirectional proxying to plugin
  - [ ] Wait for plugin exit
- [ ] **Test Phase 6**: Test server plugin proxying
  - [ ] Create mock age plugin for testing (simple echo script)
  - [ ] Start server with mock plugin in `$PATH`
  - [ ] Connect with proxy client and verify data flow
  - [ ] Test with actual `age` binary if available
  - [ ] Test error cases: plugin not found, plugin crashes

## Phase 7: Integration Testing

- [ ] **Test end-to-end workflow**:
  - [ ] Start server: `./age-plugin-agent server`
  - [ ] In new terminal, run intercept: `./age-plugin-agent intercept yubikey` (or test plugin)
  - [ ] Verify symlinks are created and in `$PATH`
  - [ ] Test age encryption/decryption with intercepted plugin
  - [ ] Verify data flows: age → proxy → server → plugin → server → proxy → age
- [ ] **Test error handling**:
  - [ ] Test with non-existent plugin name
  - [ ] Test with invalid plugin name (special characters)
  - [ ] Test with plugin binary that's not executable
  - [ ] Test with server not running
  - [ ] Test connection timeout scenarios
- [ ] **Test concurrency**:
  - [ ] Run multiple proxy clients simultaneously
  - [ ] Verify server handles concurrent connections
  - [ ] Test race conditions with rapid connect/disconnect

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
