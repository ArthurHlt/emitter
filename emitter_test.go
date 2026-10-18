package emitter

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestFlatBasic(t *testing.T) {
	ee := New(0)
	done := make(chan struct{}, 1)
	ee.On("test", ListenerFunc(func(event Event) {
		expect(t, event.Topic(), "test")
		expect(t, event.Subject(), "elem")
		done <- struct{}{}
	}))
	ee.Emit(NewEvent("test", "elem"))
	<-done
}

func TestFlatOf(t *testing.T) {
	ee := New(3)
	nbCall := 0
	mu := &sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(2)
	ee.On("test", ListenerFuncOf[string](func(e *EventOf[string]) {
		defer wg.Done()
		mu.Lock()
		nbCall++
		mu.Unlock()
		expect(t, e.Topic(), "test")
		if e.TypedSubject() != "toto" && e.TypedSubject() != "titi" {
			t.Errorf("Expected %v (type %v) - Got %v (type %v)", "elem", reflect.TypeFor[string](), e.Subject(), reflect.TypeOf(e.Subject()))
		}
	}))

	<-ee.Emit(NewEventOf[string]("test", "toto"))
	<-ee.Emit(NewEvent("test", 1))
	<-ee.Emit(NewEventOf[string]("test", "titi"))
	wg.Wait()
}

func TestBufferedBasic(t *testing.T) {
	ee := New(1)
	// ee.Use("*", OrSkip)
	ch := make(chan struct{})
	ee.On("test", ListenerFunc(func(event Event) {
		expect(t, event.Subject(), true)
		ch <- struct{}{}
	}))
	<-ee.Emit(NewEvent("test", true))
	<-ch
}

func TestOff(t *testing.T) {
	ee := New(0)
	list1 := ListenerFunc(func(event Event) {

	})
	list2 := ListenerFunc(func(event Event) {

	})
	ee.On("test", list1)
	ee.On("test", list2)

	expect(t, len(ee.Topics()), 1, "topics count")

	l := ee.Listeners("test")
	expect(t, len(l), 2, "listeners count")

	ee.Off("test")
	l = ee.Listeners("test")
	expect(t, len(l), 0, "listeners count after off")
	expect(t, len(ee.Topics()), 0, "topics count after off")
}

func TestOnOffAll(t *testing.T) {
	ee := New(0)
	ee.On("*", ListenerFunc(func(event Event) {

	}))
	l := ee.Listeners("test")
	expect(t, len(l), 1)

	ee.Off("*")
	l = ee.Listeners("test")
	expect(t, len(l), 0)
}

func TestVoid(t *testing.T) {
	ee := New(0)

	expect(t, sizeSyncMap(ee.middlewares), 0)
	ee.Use("*", Void)
	expect(t, sizeSyncMap(ee.middlewares), 1)
	ch := make(chan struct{})
	done := make(chan struct{})
	list := ListenerFunc(func(event Event) {
		done <- struct{}{}
	})
	ee.On("test", list)
	go func() {
		select {
		case <-done:
		default:
			ch <- struct{}{}
		}
	}()
	<-ee.Emit(NewEvent("test", nil))
	<-ch
	ee.Use("*")
	ee.Off("*", list)
	expect(t, sizeSyncMap(ee.middlewares), 0)
	l := ee.Listeners("*")
	expect(t, len(l), 0)
	ee.On("test", list, Void)
	// unblocked, sending will be skipped
	<-ee.Emit(NewEvent("test", nil))
}

func TestBackwardPattern(t *testing.T) {
	ee := New(0)
	ch := make(chan struct{})
	ee.On("*", ListenerFunc(func(e Event) {
		expect(t, e.Topic(), "test")
		expect(t, e.Flag(), e.Flag()|FlagOnce)
		ch <- struct{}{}
	}), Once)
	ee.Emit(NewEvent("test", nil))
	<-ch
}

func TestOnceDeliveredOnlyOnce(t *testing.T) {
	ee := New(10)
	got := make(chan any, 10)
	ee.Once("test", ListenerFunc(func(e Event) {
		got <- e.Subject()
	}))

	first := ee.Emit(NewEvent("test", 1))
	second := ee.Emit(NewEvent("test", 2))
	<-first
	<-second

	<-got
	select {
	case subject := <-got:
		t.Errorf("Once listener received a second event: %v", subject)
	case <-time.After(50 * time.Millisecond):
	}
	expect(t, len(ee.Listeners("test")), 0, "listeners count after once")
}

func TestOffWhileDelivering(t *testing.T) {
	ee := New(0)
	listener := ListenerFunc(func(e Event) {})
	ee.On("test", listener)

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			<-ee.Emit(NewEvent("test", nil))
		})
	}
	ee.Off("test", listener)
	wg.Wait()
}

func expect(t *testing.T, a any, b any, messages ...any) {
	msg := fmt.Sprint(messages...)
	if a != b {
		t.Errorf("Expected %v (type %v) - Got %v (type %v) (msg: %s)", b, reflect.TypeOf(b), a, reflect.TypeOf(a), msg)
	}
}

func sizeSyncMap(syncMap *sync.Map) int {
	var count int
	syncMap.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}
