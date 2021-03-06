package userclients

import (
	"net"
	"testing"
)

var uh = NewUserHandler()

func TestRegisterUser(t *testing.T) {
	// Fake connection for testing
	conn, _ := net.Pipe()
	defer conn.Close()
	u := User{100, conn}

	uh.registerUser(u)

	if uh.Users[100] != conn {
		t.Fatalf("User not registered")
	}
}

// TestAcceptConnections ensures that acceptConnections successfully returns net.Conn structs for TCP
// connections received from a listener.
func TestAcceptConnections(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:9099")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	conns, stop := uh.acceptConnections(l)
	defer close(stop)

	cConn, err := net.Dial("tcp", "localhost:9099")
	if err != nil {
		t.Fatal(err)
	}
	defer cConn.Close()

	sConn := <-conns
	defer sConn.Close()

	if cConn.LocalAddr().String() != sConn.RemoteAddr().String() {
		t.Fatalf(
			"Invalid connection received: %v != %v",
			cConn.LocalAddr().String(), sConn.RemoteAddr().String(),
		)
	}
}

// TestHandleUser ensures that the handleUser function correctly receives a connection and sends
// back a User over the channel.
func TestHandleUser(t *testing.T) {
	// This pipe is used to write over the connection that is fed to handleUser.
	client, server := net.Pipe()
	defer func() {
		client.Close()
		server.Close()
	}()

	ch := uh.handleUser(server)

	// Send a user ID over the connection.
	client.Write([]byte("123\n"))

	// Get a User struct back over the channel.
	u := <-ch

	// Verify the user ID is the one we sent.
	if u.id != 123 {
		t.Fatalf("Invalid user ID: got %v, want 123", u.id)
	}
}

// TODO Cover the rest of the important functions in the package.
