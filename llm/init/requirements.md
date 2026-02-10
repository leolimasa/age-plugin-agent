# Objective

Implement an age plugin that allows users to forward encryption and decryption operations to a remote server.

# Architecture

The client implements the `age plugin` protocol and forwards requests to the server. The server performs the actual encryption and decryption operations by calling the appropriate plugin (or the `age` binary directly).

## Interception

Age encryption needs a **recipient**. 

Age decryption needs an **identity**.

Either of those can be either a local file or an age plugin. If it's an age plugin, age will automatically call that plugin with the corresponding binary from the $PATH.

`age-plugin-agent` will intercept the call by **replacing** the `age-plugin-[name]` in the current environment. This is done by executing `age-plugin-agent intercept plugin1,plugin2 [shell (optional)]`, which will spin up a shell that has `age-plugin-plugin1` and `age-plugin-plugin2` in the $PATH, both pointing to `age-plugin-agent`. 

Those would be equivalent:

```bash
age-plugin-agent intercept yubikey
```

```bash
alias age-plugin-yubikey='age-plugin-agent'
```

# Implementation

* Go executable called `age-plugin-agent`.
* **Binary Name Detection**: The binary should check its own name (via `os.Args[0]`) on startup:
	* If the binary name starts with `age-plugin-` (e.g., called via symlink as `age-plugin-yubikey`), extract the plugin name and automatically run in `forward` mode for that plugin.
	* Otherwise, parse subcommands normally.
* Subcommands:
	* `intercept`: Spins up a shell with the specified plugins in the $PATH. Creates symlinks to `age-plugin-agent` named `age-plugin-<name>` for each specified plugin, allowing automatic detection via binary name.
	* `forward <plugin-name>`: Connects to the server socket, performs the handshake protocol (sends plugin name, receives OK/ERROR), then forwards STDIN to the socket and forwards responses from the socket back to STDOUT.
	* `server`: Spins up the server that listens on a Unix domain socket for plugin requests. For each connection, it performs the handshake protocol to determine which plugin to execute, searches `$PATH` for the corresponding `age-plugin-[name]` binary, spawns the plugin subprocess, and transparently proxies data between the socket and the plugin's stdin/stdout. Once the plugin binary exits, the server closes the connection.
* Socket: the socket is simply a file. The server listens on that socket for incoming plugin requests. The client connects to that socket and sends the plugin request. The server then forwards the request to the appropriate plugin and sends the response back to the client. This is meant to be used by forwarding the file via SSH.

## Handshake Protocol

Before proxying data to the plugin, the client and server perform a handshake to establish which plugin to execute and handle any initialization errors.

### Protocol Flow

1. **Client connects** to the Unix domain socket
2. **Client sends plugin name**: `<plugin-name>\n`
   - Format: UTF-8 encoded plugin name followed by newline (`\n`)
   - Example: `yubikey\n`
   - Plugin name must be alphanumeric with optional hyphens (validated by regex: `^[a-zA-Z0-9-]+$`)
   - Maximum length: 64 characters
3. **Server validates and responds**:
   - **Success case**: Server sends `OK\n` and begins transparent proxying
   - **Error case**: Server sends `ERROR <message>\n` and closes the connection
4. **After successful handshake**: Bidirectional transparent proxying begins
   - Client proxies: stdin → socket, socket → stdout
   - Server proxies: socket → plugin stdin, plugin stdout → socket
5. **Connection termination**: When plugin exits, server closes the socket connection

### Server-Side Plugin Discovery

When the server receives a plugin name:

1. Validate the plugin name format (alphanumeric + hyphens only, max 64 chars)
2. Search `$PATH` for an executable named `age-plugin-<name>`
3. Validate the binary:
   - File exists and is executable
   - Follow symlinks
   - Reject if not found or not executable
4. If validation fails, send error response and close connection
5. If validation succeeds, send `OK\n` response

### Error Messages

Error messages should be descriptive and follow this format:
- `ERROR plugin not found: <name>\n` - Plugin binary not found in PATH
- `ERROR invalid plugin name: <name>\n` - Plugin name contains invalid characters
- `ERROR plugin not executable: <path>\n` - Plugin file exists but is not executable

### Example Exchange

**Successful handshake:**
```
Client → Server: yubikey\n
Server → Client: OK\n
[transparent proxying begins]
```

**Failed handshake (plugin not found):**
```
Client → Server: nonexistent\n
Server → Client: ERROR plugin not found: nonexistent\n
[connection closes]
```

**Failed handshake (invalid name):**
```
Client → Server: ../../../etc/passwd\n
Server → Client: ERROR invalid plugin name: ../../../etc/passwd\n
[connection closes]
```
