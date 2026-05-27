package jet

import (
	"golang.org/x/exp/constraints"
	"reflect"
	"sort"
	"time"
)

type PlainType interface {
	constraints.Integer | constraints.Float | ~string | ~bool | time.Time
}

type Nillable[T any] struct {
	V *T `json:"value"` // Value
}

func NewNillable[T any](v *T) *Nillable[T] {
	return &Nillable[T]{V: v}
}

// ModifiedPlain returns the result value, and a boolean flag that shows if the value was modified or not
// can be used for plain types: integer, float, string
func ModifiedPlain[T PlainType](modified bool, cur T, update T) (T, bool) {
	if update != cur {
		return update, true
	}
	return cur, modified
}

// Modified returns result value and boolean flag that shows if the value was modified or not
// can be used for structs. For plain type use ModifiedPlain instead
func Modified[T comparable](modified bool, cur T, update T) (T, bool) {
	if !reflect.DeepEqual(cur, update) {
		return update, true
	}
	return cur, modified
}

// ModifiedPlainNillable returns result value and boolean flag that shows if the value was modified or not
// if the 'update' parameter is nil means the value has not been updated
func ModifiedPlainNillable[T PlainType](modified bool, cur T, update *T) (T, bool) {
	if updatedPlain(cur, update) {
		return *update, true
	}
	return cur, modified
}

// ModifiedNillable returns result value and boolean flag that shows if an object was modified or not
// if the 'update' parameter is nil means the value has not been updated
// []int{1,2,3} and []int{3,2,1} is NOT equal
// For plain types use ModifiedPlainNillable
func ModifiedNillable[T any](modified bool, cur *T, update *Nillable[T]) (*T, bool) {
	if updatedNillable(cur, update) {
		return update.V, true
	}
	return cur, modified
}

// ModifiedSliceNillable return result slice and boolean flag that shows if the slice was modified or not
// if the 'update' parameter is nil means value has not been updated
// if the 'update' parameter is empty means value has been updated to nil
// []int{1,2,3} and []int{3,2,1} is equal
// Side effect: result value can be sorted
func ModifiedSliceNillable[T constraints.Ordered](modified bool, cur []T, update []T) ([]T, bool) {
	if updatedSlice(sortSlice, cur, update) {
		return nilIfEmptySlice(update), true
	}
	return cur, modified
}

// ModifiedSliceStructured return result slice and boolean flag that shows if the slice was modified or not
// if the 'update' parameter is nil means value has not been updated
// if the 'update' parameter is empty means value has been updated to nil
// []int{1,2,3} and []int{3,2,1} is equal
// Side effect: result value can be sorted
func ModifiedSliceStructured[T comparable](modified bool, sort func([]T), cur []T, update []T) ([]T, bool) {
	if updatedSlice(sort, cur, update) {
		return nilIfEmptySlice(update), true
	}
	return cur, modified
}

func updatedPlain[T PlainType](cur T, update *T) bool {
	if update == nil {
		return false
	}
	return *update != cur
}

func updatedNillable[T any](cur *T, update *Nillable[T]) bool {
	if update == nil {
		return false
	}
	return !reflect.DeepEqual(cur, update.V)
}

func updatedSlice[T comparable](sort func([]T), cur []T, update []T) bool {
	if update == nil {
		return false
	}
	if len(cur) != len(update) {
		return true
	}
	update = nilIfEmptySlice(update)
	if cur != nil && update == nil {
		return true
	}
	sort(cur)
	sort(update)
	for i := range cur {
		if cur[i] != update[i] {
			return true
		}
	}
	return false
}

func isEmptySlice[T comparable](val []T) bool {
	return val != nil && len(val) == 0
}

func nilIfEmptySlice[T comparable](val []T) []T {
	if isEmptySlice(val) {
		return nil
	}
	return val
}

func sortSlice[T constraints.Ordered](slice []T) {
	sort.Slice(slice, func(i, j int) bool {
		return slice[i] < slice[j]
	})
}
