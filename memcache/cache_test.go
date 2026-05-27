package memcache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_SetGet(t *testing.T) {
	mc := NewMemCache()

	mc.Set("k", "v", DefaultTtl)

	got, ok := mc.Get("k")
	assert.True(t, ok)
	assert.Equal(t, "v", got)
}

func Test_Get_Missing(t *testing.T) {
	mc := NewMemCache()

	got, ok := mc.Get("nope")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func Test_Delete(t *testing.T) {
	mc := NewMemCache()
	mc.Set("k", 42, DefaultTtl)

	mc.Delete("k")

	_, ok := mc.Get("k")
	assert.False(t, ok, "key must be gone after Delete")
}

func Test_Delete_Missing_NoPanic(t *testing.T) {
	mc := NewMemCache()
	assert.NotPanics(t, func() { mc.Delete("nope") })
}

func Test_Set_Overwrites(t *testing.T) {
	mc := NewMemCache()
	mc.Set("k", "first", DefaultTtl)
	mc.Set("k", "second", DefaultTtl)

	got, ok := mc.Get("k")
	assert.True(t, ok)
	assert.Equal(t, "second", got)
}

func Test_Set_Expires(t *testing.T) {
	mc := NewMemCache()
	mc.Set("k", "v", 20*time.Millisecond)

	got, ok := mc.Get("k")
	assert.True(t, ok, "present before expiration")
	assert.Equal(t, "v", got)

	time.Sleep(50 * time.Millisecond)

	_, ok = mc.Get("k")
	assert.False(t, ok, "expired after ttl")
}

func Test_Set_Forever(t *testing.T) {
	mc := NewMemCache()
	mc.Set("k", "v", Forever)

	time.Sleep(20 * time.Millisecond)

	got, ok := mc.Get("k")
	assert.True(t, ok, "Forever item must not expire")
	assert.Equal(t, "v", got)
}
