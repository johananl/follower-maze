package userclients

import (
	"net"
	"testing"
)

var uh = NewUserHandler()

func TestRegisterUser(t *testing.T) {
	// Fake connection for testing
	conn, _ := net.Pipe()
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
