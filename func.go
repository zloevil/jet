package jet

// ConvertSlice is a generic function to convert one slice to another with help of converter func
func ConvertSlice[TSrc any, TRes any](src []*TSrc, converter func(*TSrc) *TRes) []*TRes {
	r := make([]*TRes, 0, len(src))
	for _, i := range src {
		r = append(r, converter(i))
	}
	return r
}

// GroupBy groups by slice by the key
func GroupBy[TItem any, TKey comparable](slice []TItem, keyFn func(TItem) TKey) map[TKey][]TItem {
	r := make(map[TKey][]TItem)
	for _, i := range slice {
		r[keyFn(i)] = append(r[keyFn(i)], i)
	}
	return r
}

func Map[TItem any, TRes any](slice []TItem, mapFn func(TItem) TRes) []TRes {
	r := make([]TRes, 0, len(slice))
	for _, i := range slice {
		r = append(r, mapFn(i))
	}
	return r
}

func Filter[TItem any](slice []TItem, filterFn func(TItem) bool) []TItem {
	r := make([]TItem, 0, len(slice))
	for _, i := range slice {
		if filterFn(i) {
			r = append(r, i)
		}
	}
	return r
}

func Reduce[TItem any, TRes any, TKey comparable](slice []TItem, grpFn func(TItem) TKey, accFn func(TItem, TRes) TRes) map[TKey]TRes {
	r := make(map[TKey]TRes)
	for _, i := range slice {
		r[grpFn(i)] = accFn(i, r[grpFn(i)])
	}
	return r
}

// SliceToMap converts a passed slice to a map by the given definition of key
// Caution!!! if specified key isn't unique for the slice, only last slice item for the given key is taken to the resulted map
func SliceToMap[TItem any, TKey comparable](slice []TItem, grpFn func(TItem) TKey) map[TKey]TItem {
	r := make(map[TKey]TItem)
	for _, i := range slice {
		r[grpFn(i)] = i
	}
	return r
}

func MapValues[TKey comparable, TItem any](m map[TKey]TItem) []TItem {
	var slice []TItem
	for _, v := range m {
		slice = append(slice, v)
	}
	return slice
}

func MapKeys[TKey comparable, TItem any](m map[TKey]TItem) []TKey {
	var slice []TKey
	for k, _ := range m {
		slice = append(slice, k)
	}
	return slice
}

func ForAll[TItem any](slice []TItem, fn func(TItem)) []TItem {
	for _, i := range slice {
		fn(i)
	}
	return slice
}

// First returns selected by condition item or default
func First[TItem any](slice []*TItem, selectFn func(*TItem) bool) *TItem {
	for _, i := range slice {
		if selectFn(i) {
			return i
		}
	}
	return GetDefault[*TItem]()
}

// GetDefault get default value for type
func GetDefault[T any]() T {
	var result T
	return result
}

func ToSet[TItem any, TKey comparable](slice []TItem, keyFn func(TItem) TKey) map[TKey]struct{} {
	r := make(map[TKey]struct{})
	for _, i := range slice {
		r[keyFn(i)] = struct{}{}
	}
	return r
}

func FromSet[TKey comparable](m map[TKey]struct{}) []TKey {
	var slice []TKey
	for key, _ := range m {
		slice = append(slice, key)
	}
	return slice
}

func LeftExclusive[T comparable](left, right []T) []T {
	set := ToSet(right, func(key T) T { return key })
	var result []T
	for _, v := range left {
		if _, exists := set[v]; !exists {
			result = append(result, v)
		}
	}
	return result
}

func ContainsIntersection(slice1, slice2 []string) bool {
	set := ToSet(slice1, func(key string) string { return key })
	for _, v := range slice2 {
		if _, exists := set[v]; exists {
			return true
		}
	}
	return false
}

func NilOrInMap[T comparable](value *T, m map[T]struct{}) bool {
	if value != nil {
		_, ok := m[*value]
		return ok
	}
	return true
}
