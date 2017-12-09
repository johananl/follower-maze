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

// TestAcceptConnections ensures that AcceptConnections successfully returns net.Conn structs for TCP
// connections received from a listener.
func TestAcceptConnections(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:9099")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	ch := make(chan net.Conn)
	go uh.AcceptConnections(l, ch)
	cConn, err := net.Dial("tcp", "localhost:9099")
	if err != nil {
		t.Fatal(err)
	}
	defer cConn.Close()

	sConn := <-ch
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
	a, b := net.Pipe()

	// Run handleUser in a goroutine.
	ch := make(chan User)
	go uh.handleUser(a, ch)

	// Send a user ID over the connection.
	b.Write([]byte("123\n"))

	// Get a User struct back over the channel.
	u := <-ch

	// Verify the user ID is the one we sent.
	if u.id != 123 {
		t.Fatalf("Invalid user ID: got %v, want 123", u.id)
	}
}
