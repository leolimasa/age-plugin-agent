# Objective

Implement an age plugin that allows users to forward encryption and decryption operations to a remote server.

# Architecture

The client implements the `age plugin` protocol and forwards requests to the server. The server performs the actual encryption and decryption operations by calling the appropriate plugin (or the `age` binary directly).

## Interception

Age encryption needs a **recipient**. 

Age decryption needs an **identity**.

Either of those can be either a local file or an age plugin. If it's an age plugin, age will automatically call that plugin with the corresponding binary from the $PATH.

`age-plugin-agent` will intercept the call by **replacing** the `age-plugin-[name]` in the current environment. This is done by executing `age-plugin-agent intercept plugin1,plugin2 [shell (optional)]`, which will spin up a shell that has `age-plugin-plugin1` and `age-plugin-plugin2` in the $PATH, both pointing to `age-plugin-agent forward [plugin-name]`. 

Those would be equivalent:

```bash
age-plugin-agent intercept yubikey
```

```bash
alias age-plugin-yubikey='age-plugin-agent forward yubikey'
```

# Implementation

* Go executable called `age-plugin-agent` that implements the `age plugin` protocol when called with no arguments.
* Subcommands:
	* `intercept`: Spins up a shell with the specified plugins in the $PATH, all pointing to `age-plugin-agent`.
	* `forward`: Forwards the plugin request to the server.
	* `server`: Spins up the server that listens for plugin requests and forwards them to the appropriate plugin.
