# ssh-server

An SSH server implemented in Go — custom handling of the SSH transport/protocol and per-connection session management.

## What it does
- Server side of the SSH protocol: handshake, authentication, channels
- Manages interactive sessions per connection
- Written in Go

## Run locally
```bash
go run .
```
