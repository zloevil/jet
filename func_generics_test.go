package jet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Map(t *testing.T) {
	assert.Equal(t, []int{2, 4, 6}, Map([]int{1, 2, 3}, func(i int) int { return i * 2 }))
	assert.Empty(t, Map([]int{}, func(i int) int { return i }))
}

func Test_Filter(t *testing.T) {
	assert.Equal(t, []int{2, 4}, Filter([]int{1, 2, 3, 4}, func(i int) bool { return i%2 == 0 }))
	assert.Empty(t, Filter([]int{1, 3}, func(i int) bool { return i%2 == 0 }))
}

func Test_GroupBy(t *testing.T) {
	got := GroupBy([]int{1, 2, 3, 4}, func(i int) int { return i % 2 })
	assert.Equal(t, []int{2, 4}, got[0])
	assert.Equal(t, []int{1, 3}, got[1])
}

func Test_Reduce(t *testing.T) {
	// sum items grouped by parity
	got := Reduce([]int{1, 2, 3, 4}, func(i int) int { return i % 2 }, func(i, acc int) int { return acc + i })
	assert.Equal(t, 6, got[0]) // 2+4
	assert.Equal(t, 4, got[1]) // 1+3
}

func Test_SliceToMap(t *testing.T) {
	got := SliceToMap([]string{"a", "bb", "ccc"}, func(s string) int { return len(s) })
	assert.Equal(t, map[int]string{1: "a", 2: "bb", 3: "ccc"}, got)
}

func Test_ConvertSlice(t *testing.T) {
	a, b := 1, 2
	got := ConvertSlice([]*int{&a, &b}, func(i *int) *int { v := *i * 10; return &v })
	assert.Len(t, got, 2)
	assert.Equal(t, 10, *got[0])
	assert.Equal(t, 20, *got[1])
}

func Test_MapValues_MapKeys(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	assert.ElementsMatch(t, []int{1, 2}, MapValues(m))
	assert.ElementsMatch(t, []string{"a", "b"}, MapKeys(m))
}

func Test_ForAll(t *testing.T) {
	sum := 0
	in := []int{1, 2, 3}
	out := ForAll(in, func(i int) { sum += i })
	assert.Equal(t, 6, sum)
	assert.Equal(t, in, out)
}

func Test_ToSet_FromSet(t *testing.T) {
	set := ToSet([]string{"a", "b", "a"}, func(s string) string { return s })
	assert.Len(t, set, 2, "duplicates collapse")
	_, ok := set["a"]
	assert.True(t, ok)
	assert.ElementsMatch(t, []string{"a", "b"}, FromSet(set))
}

func Test_NilOrInMap(t *testing.T) {
	m := map[string]struct{}{"x": {}}
	assert.True(t, NilOrInMap[string](nil, m), "nil value is allowed")

	v := "x"
	assert.True(t, NilOrInMap(&v, m), "present value")

	w := "y"
	assert.False(t, NilOrInMap(&w, m), "absent value")
}
