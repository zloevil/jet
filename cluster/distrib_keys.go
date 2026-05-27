package cluster

import "sync"

// DistributedKeys is useful when some processing happens in cluster environment (multiple replicas)
// some business keys might be processed on particular replicas
// this helps to trace which replica owns the business key
type DistributedKeys interface {
	Set(key string)
	Remove(key string)
	Check(key string) bool
}

type distrKeysImpl struct {
	sync.RWMutex
	keys map[string]struct{}
}

func NewDistributedKeys() DistributedKeys {
	return &distrKeysImpl{
		keys: make(map[string]struct{}),
	}
}

func (c *distrKeysImpl) Set(key string) {
	c.Lock()
	defer c.Unlock()
	c.keys[key] = struct{}{}
}

func (c *distrKeysImpl) Remove(key string) {
	c.Lock()
	defer c.Unlock()
	delete(c.keys, key)
}

func (c *distrKeysImpl) Check(key string) bool {
	c.RLock()
	defer c.RUnlock()
	_, ok := c.keys[key]
	return ok
}
