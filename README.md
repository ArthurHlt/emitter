# Emitter [![godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/ArthurHlt/emitter)

The emitter package implements a channel-based pubsub pattern. The design goals are to use Golang concurrency model 
instead of flat callbacks and to design a very simple API that is easy to consume.

This was a fork of https://github.com/olebedev/emitter but code heavily changed to implement listener handler pattern, 
give ability to set any type of event and finally allow generics usage. 
Groups have been removed as they are not used in this project.

## What it does?

- [sync/async event emitting](#flags)
- [predicates/middlewares](#middlewares)
- [bi-directional wildcard](#wildcard)
- [custom event](#event)


## Brief example

```go
package main

import (
 "fmt"
 "github.com/ArthurHlt/emitter"
)

func main(){
	e := emitter.New(10)
    
	// simple
    e.On("change_any", emitter.ListenerFunc(func(event emitter.Event) {
        fmt.Println(event.Topic())
        fmt.Println(event.Subject())
    }))
    <-e.Emit(emitter.NewEvent("change_any", 2)) // wait for the event sent successfully
    <-e.Emit(emitter.NewEvent("change_any", 37))
	
	// with generics
	e.On("change_any", emitter.ListenerFuncOf[string](func(event *emitter.EventOf[string]) {
        fmt.Println(event.Topic())
        fmt.Println(event.TypedSubject())
    }))
	
    e.Off("*") // unsubscribe any listeners
    // listener channel was closed
}

```

## Constructor
`emitter.New` takes a `uint` as the first argument to indicate what buffer size should be used for listeners. 
It is also possible to change the buffer capacity during runtime using the following code: `e.Cap = 10`.

## Wildcard
The package allows publications and subscriptions with wildcard. This feature is based on `path.Match` function.

Example:

```go
e.Once("*", emitter.ListenerFunc(func(event emitter.Event) {
    fmt.Println(event.Topic())
    fmt.Println(event.Subject()) // show 42 and after 37
}))
e.Emit(emitter.NewEvent("something:special", 42))
e.Emit(emitter.NewEvent("*", 37))
```

Note that the wildcard uses `path.Match`, but the lib does not return errors related to parsing for this is not the main feature. Please check the topic specifically via `emitter.Test()` function.

## Middlewares
An important part of pubsub package is the predicates. It should be allowed to skip some events. Middlewares address this problem.
The middleware is a function that takes a pointer to the `Event` as its first argument. A middleware is capable of doing the following items:

1. It allows you to modify an event.
2. It allows skipping the event emitting if needed.
3. It also allows modification of the event's arguments.
4. It allows you to specify the mode to describe how exactly an event should be emitted(see [below](#flags)).

There are two ways to add middleware into the event emitting flow:

- via .On("event", listener, middlewares...)
- via .Use("event", middlewares...)

The first one add middlewares only for a particular listener, while the second one adds middlewares for all events with a given topic.

For example:
```go
// use synchronous mode for all events, it also depends
// on the emitter capacity(buffered/unbuffered channels)
e.Use("*", emitter.Void)

e.Once("*", emitter.ListenerFunc(func(event emitter.Event) {
	// nothing will never printed
    fmt.Println(event.Topic())
    fmt.Println(event.Subject())
}))
e.Emit(emitter.NewEvent("something:special", 42))
```


## Flags
Flags needs to describe how exactly the event should be emitted. The available options are listed [here](https://godoc.org/github.com/olebedev/emitter#Flag).

Every event(`emitter.Event`) has a field called`.Flags` that contains flags as a binary mask.
Flags can be set only via middlewares(see above).

There are several predefined middlewares to set needed flags:

- [`emitter.Once`](https://godoc.org/github.com/ArthurHlt/emitter#Once)
- [`emitter.Void`](https://godoc.org/github.com/ArthurHlt/emitter#Void)
- [`emitter.Skip`](https://godoc.org/github.com/ArthurHlt/emitter#Skip)

You can chain the above flags as shown below:
```go
e.Use("*", emitter.Void) // skip sending for any events

event := <-e.On("*", emitter.ListenerFunc(func(event emitter.Event) {
            // nothing will never printed
            fmt.Println(event.Topic())
            fmt.Println(event.Subject())
        }),
        emitter.Once) // set custom flags for this listener

e.Emit(emitter.NewEvent("surprise", 65536))
```

## Event

Event is an interface used to pass events to listeners.

It contains the following methods:

```go
type Event interface {
	// Topic returns the topic of the event.
	Topic() string
	// Flag returns the flag of the event.
	Flag() Flag
	// SetFlag sets the flag of the event.
	SetFlag(Flag)
	// Subject returns the subject of the event.
	Subject() any
	// Clone returns a copy of the event.
	Clone() Event
}
```

You can provide your own event implementation by implementing the `Event` interface.

By default you have 2 implementations:
- `emitter.BasicEvent` - a simple event implementation created with `emitter.NewEvent`
- `emitter.EventOf` - a generic event implementation created with `emitter.NewEventOf`
