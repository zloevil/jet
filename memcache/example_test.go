package memcache_test

import (
	"fmt"

	"github.com/zloevil/jet/memcache"
)

func ExampleMemCache() {
	c := memcache.NewMemCache()
	c.Set("greeting", "hello", memcache.DefaultTtl)

	v, ok := c.Get("greeting")
	fmt.Println(v, ok)

	_, ok = c.Get("missing")
	fmt.Println(ok)
	// Output:
	// hello true
	// false
}
