package event

import (
	"context"
	"github.com/zloevil/jet"
	"github.com/zloevil/jet/goroutine"
	"reflect"
	"sync"
)

const (
	ErrCodeBusNotReflectType = "BUS-001"
	ErrCodeBusTopicNotExists = "BUS-002"
)

var (
	ErrBusNotReflectType = func(v reflect.Kind) error {
		return jet.NewAppErrBuilder(ErrCodeBusNotReflectType, "%s is not of type reflect.Func", v).Err()
	}
	ErrBusTopicNotExists = func(topic string) error {
		return jet.NewAppErrBuilder(ErrCodeBusTopicNotExists, "topic doesn't exists (%s)", topic).Err()
	}
)

// BusSubscriber defines subscription-related bus behavior
type BusSubscriber interface {
	Subscribe(topic string, fn interface{}) error
	SubscribeAsync(topic string, fn interface{}, transactional bool) error
	SubscribeOnce(topic string, fn interface{}) error
	SubscribeOnceAsync(topic string, fn interface{}) error
	Unsubscribe(topic string, handler interface{}) error
}

// BusPublisher defines publishing-related bus behavior
type BusPublisher interface {
	Publish(ctx context.Context, topic string, args ...interface{})
}

// BusController defines bus control behavior (checking handler's presence, synchronization)
type BusController interface {
	HasCallback(topic string) bool
	WaitAsync()
}

// Bus is a global (subscribe, publish, control) bus behavior
type Bus interface {
	BusController
	BusSubscriber
	BusPublisher
}

// eventBus - box for handlers and callbacks.
type eventBus struct {
	handlers map[string][]*busEventHandler
	lock     sync.Mutex // a lock for the map
	wg       sync.WaitGroup
	logger   jet.CLoggerFunc
}

type busEventHandler struct {
	callBack      reflect.Value
	flagOnce      bool
	async         bool
	transactional bool
	sync.Mutex    // lock for an event handler - useful for running async callbacks serially
}

// New returns new eventBus with empty handlers.
func New(l jet.CLoggerFunc) Bus {
	b := &eventBus{
		handlers: make(map[string][]*busEventHandler),
		lock:     sync.Mutex{},
		wg:       sync.WaitGroup{},
		logger:   l,
	}
	return Bus(b)
}

func (b *eventBus) l() jet.CLogger {
	return b.logger().Cmp("event-bus")
}

// doSubscribe handles the subscription logic and is utilized by the public Subscribe functions
func (b *eventBus) doSubscribe(topic string, fn interface{}, handler *busEventHandler) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	if !(reflect.TypeOf(fn).Kind() == reflect.Func) {
		return ErrBusNotReflectType(reflect.TypeOf(fn).Kind())
	}
	b.handlers[topic] = append(b.handlers[topic], handler)
	return nil
}

// Subscribe subscribes to a topic.
// Returns error if `fn` is not a function.
func (b *eventBus) Subscribe(topic string, fn interface{}) error {
	return b.doSubscribe(topic, fn, &busEventHandler{
		reflect.ValueOf(fn), false, false, false, sync.Mutex{},
	})
}

// SubscribeAsync subscribes to a topic with an asynchronous callback
// Transactional determines whether subsequent callbacks for a topic are
// run serially (true) or concurrently (false)
// Returns error if `fn` is not a function.
func (b *eventBus) SubscribeAsync(topic string, fn interface{}, transactional bool) error {
	return b.doSubscribe(topic, fn, &busEventHandler{
		reflect.ValueOf(fn), false, true, transactional, sync.Mutex{},
	})
}

// SubscribeOnce subscribes to a topic once. Handler will be removed after executing.
// Returns error if `fn` is not a function.
func (b *eventBus) SubscribeOnce(topic string, fn interface{}) error {
	return b.doSubscribe(topic, fn, &busEventHandler{
		reflect.ValueOf(fn), true, false, false, sync.Mutex{},
	})
}

// SubscribeOnceAsync subscribes to a topic once with an asynchronous callback
// Handler will be removed after executing.
// Returns error if `fn` is not a function.
func (b *eventBus) SubscribeOnceAsync(topic string, fn interface{}) error {
	return b.doSubscribe(topic, fn, &busEventHandler{
		reflect.ValueOf(fn), true, true, false, sync.Mutex{},
	})
}

// HasCallback returns true if exists any callback subscribed to the topic.
func (b *eventBus) HasCallback(topic string) bool {
	b.lock.Lock()
	defer b.lock.Unlock()
	_, ok := b.handlers[topic]
	if ok {
		return len(b.handlers[topic]) > 0
	}
	return false
}

// Unsubscribe removes callback defined for a topic.
// Returns error if there are no callbacks subscribed to the topic.
func (b *eventBus) Unsubscribe(topic string, handler interface{}) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	if _, ok := b.handlers[topic]; ok && len(b.handlers[topic]) > 0 {
		b.removeHandler(topic, b.findHandlerIdx(topic, reflect.ValueOf(handler)))
		return nil
	}
	return ErrBusTopicNotExists(topic)
}

// Publish executes callback defined for a topic. Any additional argument will be transferred to the callback.
func (b *eventBus) Publish(ctx context.Context, topic string, args ...interface{}) {
	b.lock.Lock() // will unlock if handler is not found or always after setUpPublish
	defer b.lock.Unlock()
	if handlers, ok := b.handlers[topic]; ok && len(handlers) > 0 {
		// Handlers slice may be changed by removeHandler and Unsubscribe during iteration,
		// so make a copy and iterate the copied slice.
		for i, handler := range handlers {
			handler := handler
			if handler == nil {
				continue
			}
			if handler.flagOnce {
				b.removeHandler(topic, i)
			}
			if !handler.async {
				b.doPublish(ctx, handler, topic, args...)
			} else {
				b.wg.Add(1)
				if handler.transactional {
					b.lock.Unlock()
					handler.Lock()
					b.lock.Lock()
				}
				goroutine.New().WithLogger(b.l().C(ctx).Mth("publish")).Go(ctx, func() {
					defer b.wg.Done()
					b.doPublishAsync(ctx, handler, topic, args...)
				})
			}
		}
	}
}

func (b *eventBus) doPublish(ctx context.Context, handler *busEventHandler, topic string, args ...interface{}) {
	b.l().C(ctx).Mth("do-publish").TrcF("topic: %s args: %v", topic, args)
	passedArguments := b.setUpPublish(handler, append([]any{ctx}, args...)...)
	handler.callBack.Call(passedArguments)
}

func (b *eventBus) doPublishAsync(ctx context.Context, handler *busEventHandler, topic string, args ...interface{}) {
	if handler.transactional {
		defer handler.Unlock()
	}
	b.doPublish(ctx, handler, topic, args...)
}

func (b *eventBus) removeHandler(topic string, idx int) {
	if _, ok := b.handlers[topic]; !ok {
		return
	}
	l := len(b.handlers[topic])

	if !(0 <= idx && idx < l) {
		return
	}

	v := make([]*busEventHandler, l-1)
	for i, j := 0, 0; i < len(b.handlers[topic]); i++ {
		if i == idx {
			continue
		}
		v[j] = b.handlers[topic][i]
		j++
	}
	b.handlers[topic] = v
}

func (b *eventBus) findHandlerIdx(topic string, callback reflect.Value) int {
	if _, ok := b.handlers[topic]; ok {
		for idx, handler := range b.handlers[topic] {
			if handler.callBack.Type() == callback.Type() &&
				handler.callBack.Pointer() == callback.Pointer() {
				return idx
			}
		}
	}
	return -1
}

func (b *eventBus) setUpPublish(callback *busEventHandler, args ...interface{}) []reflect.Value {
	funcType := callback.callBack.Type()
	passedArguments := make([]reflect.Value, len(args))
	for i, v := range args {
		if v == nil {
			passedArguments[i] = reflect.New(funcType.In(i)).Elem()
		} else {
			passedArguments[i] = reflect.ValueOf(v)
		}
	}

	return passedArguments
}

// WaitAsync waits for all async callbacks to complete
func (b *eventBus) WaitAsync() {
	b.wg.Wait()
}
