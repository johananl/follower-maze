package main

import (
	"reflect"
	"testing"
)

//func TestAcceptEvents(t *testing.T) {
//	// Initialize the queue manager
//	qm := NewQueueManager()
//
//	// Initialize the user handler
//	uh := NewUserHandler()
//
//	// Initialize the event handler
//	eh := NewEventHandler(qm, uh)
//
//	l, err := net.Listen("tcp", ":9999")
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer l.Close()
//
//}

var qm = NewQueueManager()
var uh = NewUserHandler()
var eh = NewEventHandler(qm, uh)

var parseEventTests = []struct {
	in  string
	out *Event
}{
	{"666|F|60|50", &Event{sequence: 666, eventType: follow, fromUserId: 60, toUserId: 50}},
	{"1|U|12|9", &Event{sequence: 1, eventType: unfollow, fromUserId: 12, toUserId: 9}},
}

func TestParseEvent(t *testing.T) {
	for _, te := range parseEventTests {
		e, err := eh.parseEvent(te.in)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(e, te.out) {
			t.Fatalf("Event parsing failed: got %v, want %v", e, te.out)
		}
	}
}
