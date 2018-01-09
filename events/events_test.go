package events

import (
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"testing"

	"bitbucket.org/johananl/follower-maze/userclients"
)

var qm = NewQueueManager()
var uh = userclients.NewUserHandler()
var eh = NewEventHandler(qm, uh)

var goodEvents = []struct {
	in  string
	out event
}{
	{"666|F|60|50\n",
		event{rawEvent: "666|F|60|50\n", sequence: 666, eventType: follow, fromUserID: 60, toUserID: 50},
	},
	{"1|U|12|9\n",
		event{rawEvent: "1|U|12|9\n", sequence: 1, eventType: unfollow, fromUserID: 12, toUserID: 9},
	},
	{"542532|B\n",
		event{rawEvent: "542532|B\n", sequence: 542532, eventType: broadcast},
	},
	{"43|P|32|56\n",
		event{rawEvent: "43|P|32|56\n", sequence: 43, eventType: privateMsg, fromUserID: 32, toUserID: 56},
	},
	{"634|S|32\n",
		event{rawEvent: "634|S|32\n", sequence: 634, eventType: statusUpdate, fromUserID: 32},
	},
}

var badEvents = []string{
	"abcd",
	"abcd\n",
	"\n",
	"634|S|",
	"666|F|60|50|",
	"666|F|60||50",
	"",
	" ",
	"(&*(^*&^$$#",
	"ばか猫",
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
		_, err := eh.parseEvent(te)
		if err == nil {
			t.Fatalf("Expected to get an error: %v should not parse", te)
		}
	}
}

var testEvent = event{
	rawEvent:   "666|F|60|50\n",
	sequence:   666,
	eventType:  follow,
	fromUserID: 60,
	toUserID:   50,
}

func TestPushEvent(t *testing.T) {
	stop := qm.Run()

	qm.pushEvent(testEvent)

	if qm.queueLength() != 1 {
		t.Fatalf("Invalid queue length after queueing event: got %d, want %d", qm.queueLength(), 1)
	}

	qm.popEvent()
	stop <- true
}

func TestPopEvent(t *testing.T) {
	stop := qm.Run()
	qm.pushEvent(testEvent)

	e := qm.popEvent()

	if e != testEvent {
		t.Fatalf("Invalid event popped from queue: got %v, want %v", e, testEvent)
	}
	if qm.queueLength() != 0 {
		t.Fatalf("Popped event not deleted from queue")
	}

	stop <- true
}

// TestQueueOrdering verifies that the queue properly orders events. It does so by generating
// a set of events, shuffling them and storing them in the queue. The events should be popped
// in the correct order (by sequence).
func TestQueueOrdering(t *testing.T) {
	numEvents := 1000

	// Populate events slice
	events := []event{}
	for i := 1; i <= numEvents; i++ {
		events = append(events, event{
			rawEvent:   fmt.Sprintf("%d|F|60|50\n", i),
			sequence:   i,
			eventType:  follow,
			fromUserID: 60,
			toUserID:   50,
		})
	}

	// Shuffle events slice
	for i := range events {
		j := rand.Intn(i + 1)
		events[i], events[j] = events[j], events[i]
	}

	stop := qm.Run()

	for _, e := range events {
		qm.pushEvent(e)
	}

	for i := 1; i <= numEvents; i++ {
		e := qm.popEvent()
		if e.sequence != i {
			t.Fatalf("Wrong sequence received from queue: got %v want %v", e.sequence, i)
		}
	}

	stop <- true
}

// TestAcceptConnections ensures that acceptConnections successfully returns net.Conn structs for TCP
// connections received from a listener.
func TestAcceptConnections(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:9090")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	ch, stop := eh.acceptConnections(l)
	defer close(stop)

	cConn, err := net.Dial("tcp", "localhost:9090")
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

// TestHandleEvents ensures that handleEvents successfully returns event structs.
func TestHandleEvents(t *testing.T) {
	client, server := net.Pipe()
	defer func() {
		client.Close()
		server.Close()
	}()

	events := eh.handleEvents(server)

	for _, te := range goodEvents {
		client.Write([]byte(te.in))
		e := <-events

		if !reflect.DeepEqual(e, te.out) {
			t.Fatalf("Wrong event received: got %v, want %v", e, te.out)
		}
	}
}

// TODO Test event processing
