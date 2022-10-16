package emitter

// Event is an interface used to pass events to listeners.
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

func NewEvent(topic string, subject any) *BasicEvent {
	return &BasicEvent{
		topic:   topic,
		subject: subject,
	}
}

type BasicEvent struct {
	topic   string
	flag    Flag
	subject any
}

func (e *BasicEvent) Topic() string {
	return e.topic
}

func (e *BasicEvent) Flag() Flag {
	return e.flag
}

func (e *BasicEvent) SetFlag(flag Flag) {
	e.flag = flag
}

func (e *BasicEvent) Subject() any {
	return e.subject
}

func (e *BasicEvent) Clone() Event {
	return &BasicEvent{
		topic:   e.topic,
		flag:    e.flag,
		subject: e.subject,
	}
}

type EventOf[T any] struct {
	topic   string
	flag    Flag
	subject T
}

func NewEventOf[T any](topic string, subject T) *EventOf[T] {
	return &EventOf[T]{
		topic:   topic,
		subject: subject,
	}
}

func (e *EventOf[T]) Topic() string {
	return e.topic
}

func (e *EventOf[T]) Flag() Flag {
	return e.flag
}

func (e *EventOf[T]) SetFlag(flag Flag) {
	e.flag = flag
}

func (e *EventOf[T]) Subject() any {
	return e.subject
}

func (e *EventOf[T]) TypedSubject() T {
	return e.subject
}

func (e *EventOf[T]) Clone() Event {
	return &EventOf[T]{
		topic:   e.topic,
		flag:    e.flag,
		subject: e.subject,
	}
}
