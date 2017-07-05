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

var goodEvents = []struct {
	in  string
	out *Event
}{
	{"666|F|60|50", &Event{sequence: 666, eventType: follow, fromUserId: 60, toUserId: 50}},
	{"1|U|12|9", &Event{sequence: 1, eventType: unfollow, fromUserId: 12, toUserId: 9}},
	{"542532|B", &Event{sequence: 542532, eventType: broadcast}},
	{"43|P|32|56", &Event{sequence: 43, eventType: privateMsg, fromUserId: 32, toUserId: 56}},
	{"634|S|32", &Event{sequence: 634, eventType: statusUpdate, fromUserId: 32}},
}

var badEvents = []string{
	"abcd",
	"634|S|",
	"666|F|60|50|",
	"",
}

func TestParseEvent(t *testing.T) {
	for _, te := range goodEvents {
		e, err := eh.parseEvent(te.in)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(e, te.out) {
			t.Fatalf("Event parsing failed: got %v, want %v", e, te.out)
		}
	}
}

func TestParseEventErrors(t *testing.T) {
	for _, te := range badEvents {
		e, err := eh.parseEvent(te)
		if e != nil || err == nil {
			t.Fatalf("Got an event instead of an error: %v should not parse", te)
		}
	}
}
