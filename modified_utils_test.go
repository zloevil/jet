package jet

import (
	"github.com/stretchr/testify/assert"
	"slices"
	"testing"
	"time"
)

type test struct {
	A int
	B string
	C *bool
}

func Test_ModifiedPlain(t *testing.T) {
	// do not update
	res, mod := ModifiedPlain(false, 1, 1)
	assert.False(t, mod)
	assert.Equal(t, 1, res)
	// change value
	res, mod = ModifiedPlain(false, 1, 2)
	assert.True(t, mod)
	assert.Equal(t, 2, res)

	now := Now()
	// do not update
	tRes, mod := ModifiedPlain(false, now, now)
	assert.False(t, mod)
	assert.Equal(t, now, tRes)
	// change value
	tRes, mod = ModifiedPlain(false, now, now.Add(time.Second))
	assert.True(t, mod)
	assert.Equal(t, now.Add(time.Second), tRes)
}

func Test_Modified(t *testing.T) {
	// do not update
	res, mod := Modified(false, 1, 1)
	assert.False(t, mod)
	assert.Equal(t, 1, res)
	// change value
	res, mod = Modified(false, 1, 2)
	assert.True(t, mod)
	assert.Equal(t, 2, res)
	// the same value
	resStruct, mod := Modified(false, test{}, test{})
	assert.False(t, mod)
	assert.Equal(t, test{}, resStruct)
	// the same value
	resStruct, mod = Modified(false, test{A: 1}, test{A: 1})
	assert.False(t, mod)
	assert.Equal(t, test{A: 1}, resStruct)
	// the same value
	resStruct, mod = Modified(false, test{A: 10, B: "100", C: BoolPtr(true)}, test{A: 10, B: "100", C: BoolPtr(true)})
	assert.False(t, mod)
	assert.Equal(t, 10, resStruct.A)
	assert.Equal(t, "100", resStruct.B)
	assert.True(t, *resStruct.C)
	// updated
	resStruct, mod = Modified(false, test{A: 10, B: "100", C: BoolPtr(false)}, test{A: 10, B: "100", C: BoolPtr(true)})
	assert.True(t, mod)
	assert.Equal(t, 10, resStruct.A)
	assert.Equal(t, "100", resStruct.B)
	assert.True(t, *resStruct.C)
}

func Test_ModifiedPlainNillable(t *testing.T) {
	// do not update
	res, mod := ModifiedPlainNillable(false, 1, nil)
	assert.False(t, mod)
	assert.Equal(t, 1, res)
	// change value
	res, mod = ModifiedPlainNillable(false, 1, IntPtr(2))
	assert.True(t, mod)
	assert.Equal(t, 2, res)
	// the same value
	res, mod = ModifiedPlainNillable(false, 2, IntPtr(2))
	assert.False(t, mod)
	assert.Equal(t, 2, res)
}

func Test_ModifiedSliceStructured(t *testing.T) {

	type test struct {
		i int
	}

	sort := func(arr []test) {
		slices.SortFunc(arr, func(a, b test) int {
			return a.i - b.i // ascending order
		})
	}

	// do not update
	res, mod := ModifiedSliceStructured(false, sort, []test{{i: 1}, {i: 2}, {i: 3}}, nil)
	assert.False(t, mod)
	assert.Equal(t, []test{{i: 1}, {i: 2}, {i: 3}}, res)
	// reset value
	res, mod = ModifiedSliceStructured(false, sort, []test{{i: 1}, {i: 2}, {i: 3}}, []test{})
	assert.True(t, mod)
	assert.Nil(t, res)
	// try to reset value, but is it already nil
	res, mod = ModifiedSliceStructured(false, sort, nil, []test{})
	assert.False(t, mod)
	assert.Nil(t, res)
	// do not update
	res, mod = ModifiedSliceStructured[test](false, sort, nil, nil)
	assert.False(t, mod)
	assert.Nil(t, res)
	// the same empty slices
	res, mod = ModifiedSliceStructured(false, sort, []test{}, []test{})
	assert.True(t, mod)
	assert.Nil(t, res)
	// nothing to update
	res, mod = ModifiedSliceStructured(false, sort, []test{{i: 3}, {i: 1}, {i: 2}}, []test{{i: 2}, {i: 3}, {i: 1}})
	assert.False(t, mod)
	assert.Equal(t, []test{{i: 1}, {i: 2}, {i: 3}}, res)
	// value was changed
	res, mod = ModifiedSliceStructured(false, sort, []test{{i: 3}, {i: 1}, {i: 2}}, []test{{i: 2}, {i: 3}, {i: 1}, {i: 1}})
	assert.True(t, mod)
	assert.Equal(t, []test{{i: 2}, {i: 3}, {i: 1}, {i: 1}}, res)
	// value was changed - the same
	res, mod = ModifiedSliceStructured(false, sort, []test{{i: 3}, {i: 1}, {i: 2}}, []test{{i: 2}, {i: 3}, {i: 2}})
	assert.True(t, mod)
	assert.Equal(t, []test{{i: 2}, {i: 2}, {i: 3}}, res)
}

func Test_ModifiedSliceNillable(t *testing.T) {
	// do not update
	res, mod := ModifiedSliceNillable(false, []int{1, 2, 3}, nil)
	assert.False(t, mod)
	assert.Equal(t, []int{1, 2, 3}, res)
	// reset value
	res, mod = ModifiedSliceNillable(false, []int{1, 2, 3}, []int{})
	assert.True(t, mod)
	assert.Nil(t, res)
	// try to reset value, but is it already nil
	res, mod = ModifiedSliceNillable(false, nil, []int{})
	assert.False(t, mod)
	assert.Nil(t, res)
	// do not update
	res, mod = ModifiedSliceNillable[int](false, nil, nil)
	assert.False(t, mod)
	assert.Nil(t, res)
	// the same empty slices
	res, mod = ModifiedSliceNillable(false, []int{}, []int{})
	assert.True(t, mod)
	assert.Nil(t, res)
	// nothing to update
	res, mod = ModifiedSliceNillable(false, []int{3, 1, 2}, []int{2, 3, 1})
	assert.False(t, mod)
	assert.Equal(t, []int{1, 2, 3}, res)
	// value was changed
	res, mod = ModifiedSliceNillable(false, []int{3, 1, 2}, []int{2, 3, 1, 1})
	assert.True(t, mod)
	assert.Equal(t, []int{2, 3, 1, 1}, res)
	// value was changed - the same
	res, mod = ModifiedSliceNillable(false, []int{3, 1, 2}, []int{2, 3, 2})
	assert.True(t, mod)
	assert.Equal(t, []int{2, 2, 3}, res)
}

func Test_Nillable_Int(t *testing.T) {
	val := 42
	n := NewNillable(&val)
	assert.Equal(t, 42, *n.V)
}

func Test_Nillable_Slice(t *testing.T) {
	val := []int{1, 2, 3}
	n := NewNillable[[]int](&val)
	assert.Equal(t, []int{1, 2, 3}, *n.V)
}

// Test for a channel type
func Test_Nillable_Chan(t *testing.T) {
	val := make(chan int, 1)
	val <- 42
	n := NewNillable(&val)
	received := <-*n.V
	assert.Equal(t, 42, received)
}

func Test_Modified_Nillable_Plain_Ptr(t *testing.T) {
	// if object is nil, means it was not modified, no need to change init value
	res, mod := ModifiedNillable(false, StringPtr("init"), nil)
	assert.False(t, mod)
	assert.Equal(t, "init", *res)
	// do not modify but return modification true flag
	res, mod = ModifiedNillable(true, StringPtr("init"), nil)
	assert.True(t, mod)
	assert.Equal(t, "init", *res)
	// set empty value
	res, mod = ModifiedNillable(false, StringPtr("init"), NewNillable(StringPtr("")))
	assert.True(t, mod)
	assert.Empty(t, *res)
	// update value
	res, mod = ModifiedNillable(false, StringPtr("init"), NewNillable(StringPtr("updated")))
	assert.True(t, mod)
	assert.Equal(t, "updated", *res)
	// try to reset value, but is it already nil
	res, mod = ModifiedNillable(false, nil, NewNillable[string](nil))
	assert.False(t, mod)
	assert.Nil(t, res)
	// do not update
	res, mod = ModifiedNillable[string](false, nil, nil)
	assert.False(t, mod)
	assert.Nil(t, res)
	res, mod = ModifiedNillable(false, StringPtr(""), NewNillable(StringPtr("")))
	assert.False(t, mod)
	assert.Empty(t, res)
	// nothing to update
	res, mod = ModifiedNillable(false, StringPtr("init"), NewNillable(StringPtr("init")))
	assert.False(t, mod)
	assert.Equal(t, "init", *res)
}

func Test_Modified_Nillable_Ptr(t *testing.T) {
	// if object is nil, means it was not modified, no need to change init value
	res, mod := ModifiedNillable(false, &test{A: 10, B: "100", C: BoolPtr(true)}, nil)
	assert.False(t, mod)
	assert.Equal(t, 10, res.A)
	assert.Equal(t, "100", res.B)
	assert.True(t, *res.C)
	// do not modify but return modification true flag
	res, mod = ModifiedNillable(true, &test{A: 10, B: "100", C: BoolPtr(true)}, nil)
	assert.True(t, mod)
	assert.Equal(t, 10, res.A)
	assert.Equal(t, "100", res.B)
	assert.True(t, *res.C)
	// set empty value
	res, mod = ModifiedNillable(false, &test{A: 10, B: "100", C: BoolPtr(true)}, NewNillable(&test{}))
	assert.True(t, mod)
	assert.Empty(t, res)
	// try to reset value, but is it already nil
	res, mod = ModifiedNillable(false, nil, NewNillable[test](nil))
	assert.False(t, mod)
	assert.Nil(t, res)
	// do not update
	res, mod = ModifiedNillable[test](false, nil, nil)
	assert.False(t, mod)
	assert.Nil(t, res)
	res, mod = ModifiedNillable(false, &test{}, NewNillable(&test{}))
	assert.False(t, mod)
	assert.Empty(t, res)
	// nothing to update
	res, mod = ModifiedNillable(false, &test{A: 10, B: "100", C: BoolPtr(true)}, NewNillable(&test{A: 10, B: "100", C: BoolPtr(true)}))
	assert.False(t, mod)
	assert.Equal(t, 10, res.A)
	assert.Equal(t, "100", res.B)
	assert.True(t, *res.C)
	// value was changed
	res, mod = ModifiedNillable(false, &test{A: 10, B: "100", C: BoolPtr(true)}, NewNillable(&test{A: 10, B: "100", C: BoolPtr(false)}))
	assert.True(t, mod)
	assert.Equal(t, 10, res.A)
	assert.Equal(t, "100", res.B)
	assert.False(t, *res.C)
	// values were changed
	res, mod = ModifiedNillable(false, &test{A: 10, B: "100", C: BoolPtr(true)}, NewNillable(&test{A: 100, B: "10", C: nil}))
	assert.True(t, mod)
	assert.Equal(t, 100, res.A)
	assert.Equal(t, "10", res.B)
	assert.Nil(t, res.C)
}
