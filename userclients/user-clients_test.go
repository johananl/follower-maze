package userclients

import (
	"testing"
	"net"
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
