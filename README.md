# ssh-server

A minimal SSH server written in Go on top of the `golang.org/x/crypto/ssh` package. It performs the real SSH transport handshake, accepts `session` channels, and runs a tiny interactive echo "shell" over each connection. It is a compact, readable demonstration of how to stand up an SSH server in Go — accept a TCP connection, complete the SSH handshake with a host key, and service channels and requests — rather than a full-featured shell server.

## Features

- Listens for TCP connections on port `2222` and upgrades each into an SSH connection.
- Loads an RSA host key from a passphrase-protected private key file (`ssh_server_key`).
- Completes the SSH server handshake via `ssh.NewServerConn`.
- Handles one connection per goroutine (concurrent connections supported).
- Accepts `session`-type channels and rejects any other channel type as unsupported.
- Discards global (out-of-band) requests.
- Answers the `shell` request, then runs a read/write loop that echoes back whatever the client sends, prefixed with `You said: `.

## How it works

### Connection lifecycle

```
net.Listen(tcp, :2222)
  └─ Accept() loop ── per connection ──► go handleConnection(conn, config)
        ssh.NewServerConn(conn, config)   → SSH handshake using the host key
        go ssh.DiscardRequests(reqs)      → ignore global requests
        for newChannel := range chans:
            if type != "session" → Reject(UnknownChannelType)
            else Accept() ──► go handleSession(channel, requests)
                for req := range requests:
                    if req.Type == "shell" && WantReply → Reply(true); break
                    else Reply(false)
                write "Welcome to the Go SSH server!"
                loop: read up to 256 bytes, write back "You said: <input>"
```

**Handshake / host key.** At startup the server reads `ssh_server_key`, parses it with `ssh.ParsePrivateKeyWithPassphrase` (passphrase `"ssh_server"`), and registers it as the host key via `config.AddHostKey`. Generate the key with:

```
ssh-keygen -t rsa -b 2048 -f ssh_server_key -N "ssh_server"
```

**Authentication.** The `ssh.ServerConfig` is currently configured with `NoClientAuth: true`, so clients connect without credentials. A password-callback implementation (checking user `user` / password `pass`) is present in the source but commented out; enabling it would switch the server to password authentication.

**Channels and sessions.** After the handshake the server ranges over the incoming channel requests. Only the `session` channel type is accepted; everything else is rejected. Each accepted session is handled in its own goroutine, which waits for the `shell` request before dropping into the echo loop.

### Concurrency model

The `main` loop blocks on `listener.Accept()` and spawns a goroutine per TCP connection (`handleConnection`). Within a connection, global-request draining runs in its own goroutine, and each accepted session channel is serviced in yet another goroutine (`handleSession`), so multiple clients and multiple sessions are handled independently.

## Tech stack

- **Go** (module `github.com/garv2003/ssh_server`, targets `go 1.23.4`).
- **`golang.org/x/crypto v0.40.0`** — the `ssh` sub-package provides the server-side SSH protocol (transport, handshake, channels, requests).
- **`golang.org/x/sys v0.34.0`** (indirect dependency).
- Standard library: `net` (TCP listener), `io`, `os`, `log`.

## Getting started

Prerequisites: Go 1.23+ and `ssh-keygen`.

```bash
# 1. Generate the host key expected by the server (passphrase must be "ssh_server")
ssh-keygen -t rsa -b 2048 -f ssh_server_key -N "ssh_server"

# 2. Run the server (listens on port 2222)
go run ./cmd
# or
go run .
```

## Usage

Connect with any SSH client (no credentials are required in the default config):

```bash
ssh -p 2222 anyuser@localhost
```

Once connected you will see the welcome banner, and anything you type is echoed back:

```
Welcome to the Go SSH server!
hello
You said: hello
```

To require a password instead, uncomment the `PasswordCallback` block in `cmd/main.go`, remove `NoClientAuth: true`, and connect as `user` with password `pass`.

## Project structure

```
ssh-server/
├── cmd/main.go     # entire server: TCP listener, SSH handshake, channel + session handling, echo loop
├── go.mod          # module definition; depends on golang.org/x/crypto
└── go.sum
```

> Note: the server expects a host key file named `ssh_server_key` in the working directory at startup; it is generated with the `ssh-keygen` command above and is not checked into the repo.
