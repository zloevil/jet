package rpc

import (
	"context"
	"github.com/zloevil/jet"
	"github.com/zloevil/jet/goroutine"
	"sync"
	"time"
)

type Request struct {
	At       time.Time
	Callback ResponseCallback
	Msg      *Message
	Ctx      context.Context
}

// RequestPool manages incoming requests with ttl and execute handler as a Request expires
type RequestPool struct {
	sync.RWMutex
	ttl                time.Duration
	rqs                map[string]*Request
	cancelCtx          context.Context
	cancelFn           context.CancelFunc
	expirationCallback Callback
	logger             jet.CLoggerFunc
}

func NewRequestPool(logger jet.CLoggerFunc, ttl time.Duration) *RequestPool {
	return &RequestPool{
		logger: logger,
		rqs:    make(map[string]*Request),
		ttl:    ttl,
	}
}

func (c *RequestPool) l() jet.CLogger {
	return c.logger().Cmp("rpc-req-pool")
}

func (c *RequestPool) SetExpirationCallback(cb Callback) {
	c.Lock()
	defer c.Unlock()
	c.expirationCallback = cb
}

func (c *RequestPool) Queue(ctx context.Context, rq *Request) {
	c.Lock()
	defer c.Unlock()
	rq.At = jet.Now()
	c.rqs[rq.Msg.RequestId] = rq
}

func (c *RequestPool) Remove(rqId string) {
	c.Lock()
	defer c.Unlock()
	delete(c.rqs, rqId)
}

func (c *RequestPool) Len() int {
	c.RLock()
	defer c.RUnlock()
	return len(c.rqs)
}

func (c *RequestPool) TryDequeue(rqId string) *Request {
	c.Lock()
	defer c.Unlock()
	if r, ok := c.rqs[rqId]; ok {
		delete(c.rqs, rqId)
		return r
	}
	return nil
}

func (c *RequestPool) Start(ctx context.Context) {
	c.cancelCtx, c.cancelFn = context.WithCancel(ctx)
	goroutine.New().WithLogger(c.l()).Go(ctx, func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// check expired requests
				expired := make(map[string]*Request)
				func() {
					now := jet.Now()
					c.Lock()
					defer c.Unlock()
					for rqId, val := range c.rqs {
						if now.After(val.At.Add(c.ttl)) {
							expired[rqId] = val
							delete(c.rqs, rqId)
						}
					}
				}()
				// execute handlers for all the expired
				if len(expired) > 0 && c.expirationCallback != nil {
					goroutine.New().WithLogger(c.l()).Go(ctx, func() {
						for _, val := range expired {
							if err := c.expirationCallback(val.Ctx, val.Msg); err != nil {
								c.l().C(val.Ctx).E(err).Err()
							}
						}
					})
				}
			case <-c.cancelCtx.Done():
				return
			}
		}
	})
}

func (c *RequestPool) Stop() {
	c.Lock()
	defer c.Unlock()
	c.rqs = nil
	if c.cancelFn != nil {
		c.cancelFn()
	}
}
