package emitter

import (
	"sync"
	"sync/atomic"
)

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
	// ch is never closed, Emit may be sending on it: closing stopped removes the listener.
	ch          chan Event
	stopped     chan struct{}
	stopOnce    sync.Once
	onceClaimed atomic.Bool
	middlewares []func(Event)
	listener    Listener
}

func newListenerManager(capacity uint, listener Listener, middlewares ...func(Event)) *listenerManager {
	lm := &listenerManager{
		ch:          make(chan Event, capacity),
		stopped:     make(chan struct{}),
		middlewares: middlewares,
		listener:    listener,
	}
	go lm.observe()
	return lm
}

func (lm *listenerManager) observe() {
	for {
		select {
		case e := <-lm.ch:
			lm.listener.Observe(e)
		case <-lm.stopped:
			lm.observeBuffered()
			return
		}
	}
}

// observeBuffered delivers events emitted before the listener was removed.
func (lm *listenerManager) observeBuffered() {
	for {
		select {
		case e := <-lm.ch:
			lm.listener.Observe(e)
		default:
			return
		}
	}
}

func (lm *listenerManager) stop() {
	lm.stopOnce.Do(func() { close(lm.stopped) })
}

func (lm *listenerManager) send(e Event, wait bool) bool {
	if wait {
		select {
		case lm.ch <- e:
			return true
		case <-lm.stopped:
			return false
		}
	}
	select {
	case lm.ch <- e:
		return true
	default:
		return false
	}
}
