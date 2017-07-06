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
