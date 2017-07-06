package events

import (
	"reflect"
	"testing"

	"bitbucket.org/johananl/follower-maze/userclients"
)

var qm = NewQueueManager()
var uh = userclients.NewUserHandler()
var eh = NewEventHandler(qm, uh)

var goodEvents = []struct {
	in  string
	out *Event
}{
	{"666|F|60|50\n",
		&Event{rawEvent: "666|F|60|50\n", sequence: 666, eventType: follow, fromUserId: 60, toUserId: 50},
	},
	{"1|U|12|9\n",
		&Event{rawEvent: "1|U|12|9\n", sequence: 1, eventType: unfollow, fromUserId: 12, toUserId: 9},
	},
	{"542532|B\n",
		&Event{rawEvent: "542532|B\n", sequence: 542532, eventType: broadcast},
	},
	{"43|P|32|56\n",
		&Event{rawEvent: "43|P|32|56\n", sequence: 43, eventType: privateMsg, fromUserId: 32, toUserId: 56},
	},
	{"634|S|32\n",
		&Event{rawEvent: "634|S|32\n", sequence: 634, eventType: statusUpdate, fromUserId: 32},
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
}

func TestParseEvent(t *testing.T) {
	for _, te := range goodEvents {
		e, err := eh.ParseEvent(te.in)
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
		e, err := eh.ParseEvent(te)
		if e != nil || err == nil {
			t.Fatalf("Got an event instead of an error: %v should not parse", te)
		}
	}
}
