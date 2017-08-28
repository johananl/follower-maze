# Follower Maze - Solution

## General

This application is a **TCP socket server** implemented in Go. It is a proposed solution to SoundCloud's **backend
developer challenge**. The server receives an unordered stream of events in a custom text-based protocol from one _event
source_ over a TCP connection and forwards them to multiple _user clients_ according to a specific routing logic in an
ordered manner.

## Design

I have chosen to implement the solution in Go. Go is an ideal programming language for writing servers since it provides
built-in mechanisms for concurrency (goroutines, channels). In addition, Go is a low-level language which allows for
high performance. I have decided to challenge myself with this language choice: I learned Go only a couple of months
ago. My main language is Python. Still, I wanted implement the solution in Go both in order to improve my understanding
of the language and because it is just perfect for the task.

The solution is implemented using Go's standard library alone. No 3rd-party libraries were used in the implementation.
It is composed of two Go packages: **events** and **userclients**.

### The **events** Package

The events package takes care of the _event source_. It is responsible for accepting TCP connections, parsing the
messages and storing them in a **priority queue** based on their _sequence_ field. The queue is required because the
events arrive at the server at a random order while the _user clients_ expect ordered events.

The priority queue is implemented using a **min heap** data structure. This data structure is very useful here since it
provides efficient sorting upon insertion as well as retrieval of elements at a constant time (**O(1)** time
complexity). The queue has been built by implementing the _heap.Interface_ interface from the standard Go library.

### The **userclients** Package

The userclients package takes care of the _user clients_. It is responsible for handling multiple TCP connections
concurrently, registering users by associating their ID with a connection, marking Follow and Unfollow operations and
sending events to the _user clients_.

## Time Constraints and Prioritization

Disclaimer: I wrote this solution during a busy workweek in a full-time position. Therefore, I could not complete
everything I initially planned to do and had to prioritize. Following is a list of the things I wanted to improve but
didn't manage to finish in time:

- Coupling and testability - The solution isn't as loosely-coupled as I would have liked. Many of the functions require
further refactoring to make them more isolated and testable. Given more time I would have replaced some of the concrete
type arguments with interfaces to allow easy replacement in unit tests and more generic code, and would have made the
functions more isolated, possibly by breaking them down to smaller, single-responsibility functions.
- Test coverage - Unfortunately I didn't have enough time to write sufficient unit tests. I wrote some sample tests in
_events_test.go_ to illustrate the way I write tests, however I had to leave a lot of the code uncovered.
- Logging - At the moment the server's logging is pretty basic. It is good enough for figuring out when things go wrong
and for tracking the server's actions, but given more time I would have liked to improve this aspect as well.

## Building, Testing and Running

I wrote this solution on **macOS 64-bit** and tested it on **Ubuntu 16.04 LTS 64-bit** as well. I can guarantee the
solution compiles successfully on **go1.8.3 darwin/amd64**, but it is very likely to compile on other versions /
platforms without a problem.

### Building

In order to build the solution, please copy the `follower-maze` directory to `$GOPATH/src/bitbucket.org/johananl/`, `cd`
to that directory and run `go build`.

### Testing

In order to run the unit tests, please run `go test $(go list ./...)` in the project's root directory.

### Running

To run the solution after building, simply execute `./follower-maze`.

## Caveats and Limitations

### One Event Source
The solution supports **one event source**. Even though the main loop in `events.AcceptEvents()` supports multiple
_event source_ connections, I haven't implemented concurrency mechanisms (mainly locking) for more than one event
source.

### Restart After Running

Running the testing client (JAR file) more than once against my solution currently requires restarting the server. The
reason is that I haven't implemented cleanup of the data structures responsible for tracking connected users and
followers state.
