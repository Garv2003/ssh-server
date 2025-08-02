package main

import (
	"io"
	"log"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
)

// ssh-keygen -t rsa -b 2048 -f ssh_server_key -N "ssh_server"

const (
	username = "user"
	password = "pass"
	port     = "2222"
)

func main() {
	privateBytes, err := os.ReadFile("ssh_server_key")
	if err != nil {
		log.Fatal("Failed to load private key:", err)
	}
	private, err := ssh.ParsePrivateKeyWithPassphrase(privateBytes, []byte("ssh_server"))
	if err != nil {
		log.Fatal("Failed to parse private key with passphrase:", err)
	}

	config := &ssh.ServerConfig{
		NoClientAuth: true,
		//PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
		//	if c.User() == username && string(pass) == password {
		//		return nil, nil
		//	}
		//	return nil, fmt.Errorf("password rejected for %q", c.User())
		//},
		//return nil, nil
	}
	config.AddHostKey(private)

	// Listen on port
	listener, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}
	log.Printf("Listening on %s...", port)

	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept incoming connection: %v", err)
			continue
		}

		go handleConnection(tcpConn, config)
	}
}

func handleConnection(conn net.Conn, config *ssh.ServerConfig) {
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("SSH handshake failed: %v", err)
		return
	}
	defer sshConn.Close()

	log.Printf("New SSH connection from %s", sshConn.RemoteAddr())

	// Discard global requests
	go ssh.DiscardRequests(reqs)

	// Handle channels
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unsupported channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("Could not accept channel: %v", err)
			continue
		}

		go handleSession(channel, requests)
	}
}

func handleSession(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()

	for req := range requests {
		if req.Type == "shell" && req.WantReply {
			req.Reply(true, nil)
			break
		}
		req.Reply(false, nil)
	}

	io.WriteString(channel, "Welcome to the Go SSH server!\n")
	buf := make([]byte, 256)
	for {
		n, err := channel.Read(buf)
		if err != nil {
			break
		}
		channel.Write([]byte("You said: " + string(buf[:n])))
	}
}
