package jet

import (
	"github.com/stretchr/testify/assert"
	"image"
	"testing"
)

func Test_GetDefault(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		assert.Equal(t, "", GetDefault[string]())
	})
	t.Run("int", func(t *testing.T) {
		assert.Equal(t, 0, GetDefault[int]())
	})
	t.Run("float64", func(t *testing.T) {
		assert.Equal(t, 0., GetDefault[float64]())
	})
	t.Run("image.Point", func(t *testing.T) {
		assert.Equal(t, image.Point{}, GetDefault[image.Point]())
	})
	t.Run("*float64", func(t *testing.T) {
		assert.True(t, GetDefault[*float64]() == nil)
	})
}

func Test_First(t *testing.T) {
	t.Run("return value", func(t *testing.T) {
		slice := []*int{IntPtr(1), IntPtr(2), IntPtr(3)}
		assert.Equal(t, slice[1], First(slice, func(i *int) bool {
			return *i == 2
		}))
	})
	t.Run("nil slice", func(t *testing.T) {
		assert.True(t, First(nil, func(i *int) bool {
			return *i == 2
		}) == nil)
	})
	t.Run("empty slice", func(t *testing.T) {
		assert.True(t, First([]*int{}, func(i *int) bool {
			return *i == 2
		}) == nil)
	})
	t.Run("nil if not found", func(t *testing.T) {
		assert.True(t, First([]*int{IntPtr(1), IntPtr(1), IntPtr(3)}, func(i *int) bool {
			return *i == 2
		}) == nil)
	})
}

func Test_ContainsIntersection(t *testing.T) {
	tests := []struct {
		name    string
		slice1  []string
		slice2  []string
		expects bool
	}{
		{
			name:    "both empty",
			slice1:  []string{},
			slice2:  []string{},
			expects: false,
		},
		{
			name:    "first empty",
			slice1:  []string{},
			slice2:  []string{"a", "b"},
			expects: false,
		},
		{
			name:    "second empty",
			slice1:  []string{"a", "b"},
			slice2:  []string{},
			expects: false,
		},
		{
			name:    "no intersection",
			slice1:  []string{"a", "b", "c"},
			slice2:  []string{"d", "e", "f"},
			expects: false,
		},
		{
			name:    "intersection exists",
			slice1:  []string{"a", "b", "c"},
			slice2:  []string{"x", "b", "y"},
			expects: true,
		},
		{
			name:    "multiple intersections",
			slice1:  []string{"1", "2", "3", "4"},
			slice2:  []string{"4", "2"},
			expects: true,
		},
		{
			name:    "case sensitive check",
			slice1:  []string{"A", "B"},
			slice2:  []string{"a", "b"},
			expects: false,
		},
		{
			name:    "nil slice1",
			slice1:  nil,
			slice2:  []string{"a", "b"},
			expects: false,
		},
		{
			name:    "nil slice2",
			slice1:  []string{"a", "b"},
			slice2:  nil,
			expects: false,
		},
		{
			name:    "both slices nil",
			slice1:  nil,
			slice2:  nil,
			expects: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expects, ContainsIntersection(tt.slice1, tt.slice2))
		})
	}
}

func Test_LeftExclusive(t *testing.T) {
	tests := []struct {
		name    string
		left    []string
		right   []string
		expects []string
	}{
		{
			name:    "both empty",
			left:    []string{},
			right:   []string{},
			expects: nil,
		},
		{
			name:    "left empty",
			left:    []string{},
			right:   []string{"a", "b"},
			expects: nil,
		},
		{
			name:    "right empty",
			left:    []string{"a", "b"},
			right:   []string{},
			expects: []string{"a", "b"},
		},
		{
			name:    "no intersection",
			left:    []string{"a", "b", "c"},
			right:   []string{"d", "e", "f"},
			expects: []string{"a", "b", "c"},
		},
		{
			name:    "intersection exists",
			left:    []string{"a", "b", "c"},
			right:   []string{"x", "b", "y"},
			expects: []string{"a", "c"},
		},
		{
			name:    "multiple intersections",
			left:    []string{"1", "2", "3", "4"},
			right:   []string{"4", "2"},
			expects: []string{"1", "3"},
		},
		{
			name:    "multiple intersections with repeats",
			left:    []string{"4", "1", "2", "2", "3", "2", "1", "4"},
			right:   []string{"4", "2"},
			expects: []string{"1", "3", "1"},
		},
		{
			name:    "case sensitive check",
			left:    []string{"A", "B"},
			right:   []string{"a", "b"},
			expects: []string{"A", "B"},
		},
		{
			name:    "nil left",
			left:    nil,
			right:   []string{"a", "b"},
			expects: nil,
		},
		{
			name:    "nil right",
			left:    []string{"a", "b"},
			right:   nil,
			expects: []string{"a", "b"},
		},
		{
			name:    "both slices nil",
			left:    nil,
			right:   nil,
			expects: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expects, LeftExclusive(tt.left, tt.right))
		})
	}
}
