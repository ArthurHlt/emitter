/*
Package emitter implements channel based pubsub pattern.
The design goals are:
  - fully functional and safety
  - simple to understand and use
  - make the code readable, maintainable and minimalistic
*/
package emitter

import (
	"fmt"
	"path"
	"sync"
)

// Flag used to describe what behavior
// do you expect.
type Flag int

const (
	// FlagOnce indicates to remove the listener after first sending.
	FlagOnce Flag = 1 << iota
	// FlagVoid indicates to skip sending.
	FlagVoid
	// FlagSkip indicates to skip sending if channel is blocked.
	FlagSkip
)

// Middlewares.

// Once middleware sets FlagOnce flag for an event
func Once(e Event) { e.SetFlag(e.Flag() | FlagOnce) }

// Void middleware sets FlagVoid flag for an event
func Void(e Event) { e.SetFlag(e.Flag() | FlagVoid) }

// Skip middleware sets FlagSkip flag for an event
func Skip(e Event) { e.SetFlag(e.Flag() | FlagSkip) }

// New returns just created Emitter struct. Capacity argument
// will be used to create channels with given capacity
func New(capacity uint) *Emitter {
	return &Emitter{
		Cap:         capacity,
		listMans:    &sync.Map{},
		middlewares: &sync.Map{},
	}
}

// Emitter is a struct that allows to emit, receive
// event, close receiver channel, get info
// about topics and listMans
type Emitter struct {
	Cap         uint
	listMans    *sync.Map // sync.Map(string, sync.Map(string ptr addr, *listenerManager)
	middlewares *sync.Map // sync.Map(string, []func(Event))
}

// Use registers middlewares for the pattern.
func (e *Emitter) Use(pattern string, middlewares ...func(Event)) {
	if len(middlewares) == 0 {
		e.middlewares.Delete(pattern)
		return
	}
	e.middlewares.Store(pattern, middlewares)
}

// On returns a channel that will receive events. As optional second
// argument it takes middlewares.
func (e *Emitter) On(topic string, listener Listener, middlewares ...func(Event)) {
	l := newListenerManager(e.Cap, listener, middlewares...)
	rawMan, _ := e.listMans.LoadOrStore(topic, &sync.Map{})
	rawMan.(*sync.Map).Store(fmt.Sprintf("%p", listener), l)
}

// Once works exactly like On(see above) but with `Once` as the first middleware.
func (e *Emitter) Once(topic string, listener Listener, middlewares ...func(Event)) {
	e.On(topic, listener, append(middlewares, Once)...)
}

// Off unsubscribes all listMans which were covered by
// topic, it can be pattern as well.
func (e *Emitter) Off(topic string, listeners ...Listener) {
	match, _ := e.matched(topic)

	for _, _topic := range match {
		e.listMans.Range(func(topicRaw, smRaw interface{}) bool {
			if topicRaw.(string) != _topic {
				return true
			}
			if len(listeners) == 0 {
				dropAll(smRaw.(*sync.Map))
				e.listMans.Delete(topicRaw)
				return true
			}
			dropMany(smRaw.(*sync.Map), listeners...)
			shouldRemove := true
			smRaw.(*sync.Map).Range(func(_, _ interface{}) bool {
				shouldRemove = false
				return false
			})
			if shouldRemove {
				e.listMans.Delete(topicRaw)
			}
			return true
		})
	}
}

// Listeners returns slice of listMans which were covered by
// topic(it can be pattern) and error if pattern is invalid.
func (e *Emitter) Listeners(topic string) []Listener {
	match, _ := e.matched(topic)

	listeners := make([]Listener, 0)

	for _, _topic := range match {
		e.listMans.Range(func(topicRaw, smRaw interface{}) bool {
			if topicRaw.(string) != _topic {
				return true
			}
			smRaw.(*sync.Map).Range(func(_, lRaw interface{}) bool {
				listeners = append(listeners, lRaw.(*listenerManager).listener)
				return true
			})
			return true
		})
	}

	return listeners
}

// Topics returns all existing topics.
func (e *Emitter) Topics() []string {
	acc := make([]string, 0)
	e.listMans.Range(func(topicRaw, _ interface{}) bool {
		acc = append(acc, topicRaw.(string))
		return true
	})
	return acc
}

// Emit emits an event with the rest arguments to all
// listMans which were covered by topic(it can be pattern).
func (e *Emitter) Emit(event Event) chan struct{} {
	done := make(chan struct{}, 1)

	match, _ := e.matched(event.Topic())

	var wg sync.WaitGroup
	var haveToWait bool
	for _, _topic := range match {
		applyMiddlewares(event, e.getMiddlewares(_topic))
		e.listMans.Range(func(topicRaw, smRaw interface{}) bool {
			if topicRaw.(string) != _topic {
				return true
			}
			smRaw.(*sync.Map).Range(func(_, lRaw interface{}) bool {
				listMan := lRaw.(*listenerManager)
				evn := event.Clone()
				applyMiddlewares(evn, listMan.middlewares)
				if (evn.Flag() | FlagVoid) == evn.Flag() {
					return true
				}
				wg.Add(1)
				haveToWait = true
				go func(lm *listenerManager, event Event) {
					_, remove, _ := pushEvent(done, lm.ch, event)
					if remove {
						defer e.Off(event.Topic(), lm.listener)
					}
					wg.Done()
				}(listMan, evn)
				return true
			})
			return true
		})
	}
	if haveToWait {
		go func(done chan struct{}) {
			wg.Wait()
			close(done)
		}(done)
	} else {
		close(done)
	}
	return done
}

func pushEvent(
	done chan struct{},
	lstnr chan Event,
	event Event,
) (success, remove bool, err error) {
	// unwind the flags
	isOnce := (event.Flag() | FlagOnce) == event.Flag()
	isSkip := (event.Flag() | FlagSkip) == event.Flag()

	sent, canceled := send(
		done,
		lstnr,
		event,
		!isSkip,
	)
	success = sent

	if !sent && !canceled {
		remove = false
		// if not sent
	} else if !canceled {
		// if event was sent successfully
		remove = isOnce
	}
	return
}

func (e *Emitter) getMiddlewares(topic string) []func(Event) {
	var acc []func(Event)
	e.middlewares.Range(func(patternRaw, middlewaresRaw interface{}) bool {
		pattern := patternRaw.(string)
		middlewares := middlewaresRaw.([]func(Event))
		if ok, _ := path.Match(pattern, topic); ok {
			acc = append(acc, middlewares...)
		} else if ok, _ := path.Match(topic, pattern); ok {
			acc = append(acc, middlewares...)
		}
		return true
	})
	return acc
}

func applyMiddlewares(e Event, fns []func(Event)) {
	for i := range fns {
		fns[i](e)
	}
}

func (e *Emitter) matched(topic string) ([]string, error) {
	acc := make([]string, 0)
	var err error
	e.listMans.Range(func(topicRaw, _ interface{}) bool {
		curTopic := topicRaw.(string)
		var matched bool
		matched, err = path.Match(topic, curTopic)
		if err != nil {
			return false
		}
		if matched {
			acc = append(acc, curTopic)
			return true
		}
		if matched, _ = path.Match(curTopic, topic); matched {
			acc = append(acc, curTopic)
		}
		return true
	})
	return acc, err
}

func dropAll(smListeners *sync.Map) {
	smListeners.Range(func(key, value interface{}) bool {
		listMan := value.(*listenerManager)
		close(listMan.ch)
		smListeners.Delete(key)
		return true
	})
}

func dropMany(smListeners *sync.Map, listeners ...Listener) {
	for _, l := range listeners {
		drop(smListeners, l)
	}
}

func drop(smListeners *sync.Map, listener Listener) {
	ptrAddr := fmt.Sprintf("%p", listener)
	lmRaw, ok := smListeners.Load(ptrAddr)
	if !ok {
		return
	}
	listMan := lmRaw.(*listenerManager)
	close(listMan.ch)
	smListeners.Delete(ptrAddr)
}

func send(
	done chan struct{},
	ch chan Event,
	e Event, wait bool,
) (sent, canceled bool) {

	defer func() {
		if r := recover(); r != nil {
			canceled = false
			sent = false
		}
	}()

	if !wait {
		select {
		case <-done:
			break
		case ch <- e:
			sent = true
			return
		default:
			return
		}

	} else {
		select {
		case <-done:
			break
		case ch <- e:
			sent = true
			return
		}

	}
	canceled = true
	return
}
