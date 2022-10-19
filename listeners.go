package emitter

type Listener interface {
	Observe(Event)
}

type ListenerFunc func(Event)

func (fn ListenerFunc) Observe(e Event) {
	fn(e)
}

type ListenerFuncOf[T any] func(*EventOf[T])

func (fn ListenerFuncOf[T]) Observe(e Event) {
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
		lm.listener.Observe(e)
	}
}
