package jet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_MillisFromTime_TimeFromMillis_RoundTrip(t *testing.T) {
	tm := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	ms := MillisFromTime(tm)
	assert.Equal(t, tm.UnixMilli(), ms)
	assert.True(t, TimeFromMillis(ms).Equal(tm), "round trip preserves the instant")
}

func Test_NowMillis_NowNanos(t *testing.T) {
	assert.Positive(t, NowMillis())
	assert.Positive(t, NowNanos())
	assert.Greater(t, NowNanos(), NowMillis(), "nanos are larger than millis for the same instant")
}

func Test_Diff(t *testing.T) {
	a := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	b := time.Date(2025, 2, 3, 4, 5, 6, 0, time.UTC)

	y, mo, d, h, mi, s := Diff(a, b)
	assert.Equal(t, 1, y)
	assert.Equal(t, 1, mo)
	assert.Equal(t, 2, d)
	assert.Equal(t, 4, h)
	assert.Equal(t, 5, mi)
	assert.Equal(t, 6, s)
}

func Test_Diff_Symmetric(t *testing.T) {
	a := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	b := time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC)

	// order does not matter: Diff swaps so the result is the absolute difference
	y1, mo1, d1, h1, mi1, s1 := Diff(a, b)
	y2, mo2, d2, h2, mi2, s2 := Diff(b, a)
	assert.Equal(t, []int{y1, mo1, d1, h1, mi1, s1}, []int{y2, mo2, d2, h2, mi2, s2})
	assert.Equal(t, 1, h1)
}
