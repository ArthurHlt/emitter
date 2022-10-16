package emitter

type Listener interface {
	Handle(Event)
}

type ListenerFunc func(Event)

func (fn ListenerFunc) Handle(e Event) {
	fn(e)
}

type ListenerFuncOf[T any] func(*EventOf[T])

func (fn ListenerFuncOf[T]) Handle(e Event) {
	typedEvent, ok := e.(*EventOf[T])
	if !ok {
		return
	}
	fn(typedEvent)
}

type listenerManager struct {
	ch          chan Event
	middlewares []func(Event)
	listener    Listener
}

func newListenerManager(capacity uint, listener Listener, middlewares ...func(Event)) *listenerManager {
	lm := &listenerManager{
		ch:          make(chan Event, capacity),
		middlewares: middlewares,
		listener:    listener,
	}
	go func() {
		lm.observe()
	}()
	return lm
}

func (lm *listenerManager) observe() {
	for e := range lm.ch {
		lm.listener.Handle(e)
	}
}
