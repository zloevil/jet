package jet

import (
	"encoding/json"
	"fmt"
	"github.com/iancoleman/strcase"
	"math"
	"reflect"
	"strconv"
	"sync"
	"time"
)

const (
	ErrCodeJsonEncode = "TP-001"
	ErrCodeJsonDecode = "TP-002"
)

var (
	ErrJsonEncode = func(cause error) error {
		return NewAppErrBuilder(ErrCodeJsonEncode, "encode JSON").Wrap(cause).Err()
	}
	ErrJsonDecode = func(cause error) error {
		return NewAppErrBuilder(ErrCodeJsonDecode, "decode JSON").Wrap(cause).Err()
	}
)

func MapsEqual(m1, m2 map[string]interface{}) bool {
	return Equal(m1, m2)
}

func Equal(m1, m2 any) bool {
	return reflect.DeepEqual(m1, m2)
}

func MapToLowerCamelKeys(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	r := make(map[string]interface{}, len(m))
	for k, v := range m {
		if vMap, ok := v.(map[string]interface{}); ok && len(vMap) > 0 {
			r[strcase.ToLowerCamel(k)] = MapToLowerCamelKeys(vMap)
		} else {
			r[strcase.ToLowerCamel(k)] = v
		}
	}
	return r
}

func MapInterfacesToBytes(m map[string]interface{}) []byte {
	bytes, _ := json.Marshal(m)
	return bytes
}

func BytesToMapInterfaces(bytes []byte) map[string]interface{} {
	mp := make(map[string]interface{})
	_ = json.Unmarshal(bytes, &mp)
	return mp
}

func StringsToInterfaces(sl []string) []interface{} {
	if sl == nil {
		return nil
	}
	res := make([]interface{}, len(sl))
	for index, value := range sl {
		res[index] = value
	}

	return res
}

func ParseFloat32(s string) *float32 {
	if s == "" {
		return nil
	}
	fl64, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return nil
	}
	fl32 := float32(fl64)
	return &fl32
}

func ParseFloat64(s string) *float64 {
	if s == "" {
		return nil
	}
	fl64, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &fl64
}

func Round100(value float64) float64 {
	return math.Round(value*100) / 100
}

func Round10000(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func IntToInt32Ptr(i *int) *int32 {
	if i == nil {
		return nil
	}
	v := int32(*i)
	return &v
}

func IntToInt64Ptr(i *int) *int64 {
	if i == nil {
		return nil
	}
	v := int64(*i)
	return &v
}

func Int32ToIntPtr(i *int32) *int {
	if i == nil {
		return nil
	}
	v := int(*i)
	return &v
}

func UInt64ToInt32Ptr(i *uint64) *int32 {
	if i == nil {
		return nil
	}
	v := int32(*i)
	return &v
}

func Int64ToIntPtr(i *int64) *int {
	if i == nil {
		return nil
	}
	v := int(*i)
	return &v
}

func IntPtr(i int) *int {
	return &i
}

func UInt32Ptr(i uint32) *uint32 {
	return &i
}

func Float32Ptr(i float32) *float32 {
	return &i
}

func Float64Ptr(i float64) *float64 {
	return &i
}

func TimePtr(t time.Time) *time.Time {
	return &t
}

func StringPtr(s string) *string {
	return &s
}

func NowPtr() *time.Time {
	return TimePtr(Now())
}

func BoolPtr(b bool) *bool {
	return &b
}

// JsonEncode encodes type to json bytes
func JsonEncode(v any) ([]byte, error) {
	r, err := json.Marshal(&v)
	if err != nil {
		return nil, ErrJsonEncode(err)
	}
	return r, nil
}

// JsonDecode decodes type from json bytes
func JsonDecode[T any](payload []byte) (*T, error) {
	if len(payload) == 0 {
		return nil, nil
	}
	var res T
	err := json.Unmarshal(payload, &res)
	if err != nil {
		return nil, ErrJsonDecode(err)
	}
	return &res, nil
}

// JsonDecodeSlice decodes type from json bytes to slice
func JsonDecodeSlice[T any](payload []byte) ([]*T, error) {
	if len(payload) == 0 {
		return nil, nil
	}
	res, err := JsonDecodePlainSlice[T](payload)
	if err != nil {
		return nil, err
	}
	return ToSlicePtr[T](res), nil
}

// JsonDecodePlainSlice decodes type from json bytes to slice
func JsonDecodePlainSlice[T any](payload []byte) ([]T, error) {
	if len(payload) == 0 {
		return nil, nil
	}
	var res []T
	err := json.Unmarshal(payload, &res)
	if err != nil {
		return nil, ErrJsonDecode(err)
	}
	return res, nil
}

func ConvertMapValues[T any](m map[string]interface{}, converter func(value any) T) map[string]T {
	res := make(map[string]T, len(m))
	for k, v := range m {
		res[k] = converter(v)
	}
	return res
}

func ConvertFromMap[T any](data map[string]interface{}) (*T, error) {
	return ConvertFromAny[T](data)
}

func ConvertFromAny[T any](data any) (*T, error) {
	payload, err := JsonEncode(data)
	if err != nil {
		return nil, err
	}
	return JsonDecode[T](payload)
}

func ConvertToMap(data any) (map[string]interface{}, error) {
	payload, err := JsonEncode(data)
	if err != nil {
		return nil, err
	}
	decoded, err := JsonDecode[map[string]interface{}](payload)
	if err != nil {
		return nil, err
	}
	return *decoded, nil
}

// ToSlicePtr convert slice to slice ptr
func ToSlicePtr[T any](src []T) []*T {
	r := make([]*T, 0, len(src))
	for i := range src {
		r = append(r, &src[i])
	}
	return r
}

// IsEmpty gets whether the specified object is considered empty or not
func IsEmpty(object interface{}) bool {

	// get nil case out of the way
	if object == nil {
		return true
	}

	objValue := reflect.ValueOf(object)

	switch objValue.Kind() {
	// collection types are empty when they have no element
	case reflect.Chan, reflect.Map, reflect.Slice:
		return objValue.Len() == 0
	// pointers are empty if nil or if the value they point to is empty
	case reflect.Ptr:
		if objValue.IsNil() {
			return true
		}
		deref := objValue.Elem().Interface()
		return IsEmpty(deref)
	// for all other types, compare against the zero value
	// array types are empty when they match their zero-initialized state
	default:
		zero := reflect.Zero(objValue.Type())
		return reflect.DeepEqual(object, zero.Interface())
	}
}

type SafeMap[TK comparable, TV any] struct {
	m map[TK]TV
	sync.RWMutex
}

func NewSafeMap[TK comparable, TV any]() *SafeMap[TK, TV] {
	return &SafeMap[TK, TV]{
		m: make(map[TK]TV),
	}
}

func (m *SafeMap[TK, TV]) Get(key TK) TV {
	m.RLock()
	defer m.RUnlock()
	return m.m[key]
}

func (m *SafeMap[TK, TV]) TryGet(key TK) (TV, bool) {
	m.RLock()
	defer m.RUnlock()
	v, ok := m.m[key]
	return v, ok
}

func (m *SafeMap[TK, TV]) Set(key TK, val TV) {
	m.Lock()
	defer m.Unlock()
	m.m[key] = val
}

func (m *SafeMap[TK, TV]) Delete(key TK) {
	m.Lock()
	defer m.Unlock()
	delete(m.m, key)
}

// Map returns a copy of the map (it doesn't deep copy)
func (m *SafeMap[TK, TV]) Map() map[TK]TV {
	m.RLock()
	defer m.RUnlock()
	r := make(map[TK]TV, len(m.m))
	for k, v := range m.m {
		r[k] = v
	}
	return r
}

// PaginateSlice paginates the given slice of items according to the provided paging request.
// Returns a slice containing a subset of the original items based on the specified page size and index.
func PaginateSlice[T any](items []T, paging PagingRequest) []T {

	if paging.Size <= 0 {
		paging.Size = 100
	}
	if paging.Index <= 0 {
		paging.Index = 1
	}

	if len(items) == 0 {
		return []T{}
	}

	// Calculate start and end indices for the page
	start := (paging.Index - 1) * paging.Size
	end := start + paging.Size

	// If the start index is greater than the length of the slice, return an empty slice
	if start >= len(items) {
		return []T{}
	}

	// Adjust the end index if it goes beyond the slice length
	if end > len(items) {
		end = len(items)
	}

	// Return the paginated slice
	return items[start:end]
}

type TFlags interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64
}

type Flags[T TFlags] struct {
	value T
}

// HasAll checks if all specified flags are set
func (f Flags[T]) HasAll(flags ...T) bool {
	for _, flag := range flags {
		if f.value&flag != flag {
			return false
		}
	}
	return true
}

// HasAny checks if any of the specified flags are set
func (f Flags[T]) HasAny(flags ...T) bool {
	if len(flags) == 0 {
		return false
	}

	for _, flag := range flags {
		if f.value&flag != 0 {
			return true
		}
	}
	return false
}

// Set sets the specified flags and returns the new value
func (f Flags[T]) Set(flags ...T) Flags[T] {
	result := f.value
	for _, flag := range flags {
		result |= flag
	}
	return Flags[T]{value: result}
}

func NewFlags[T TFlags](flags ...T) Flags[T] {
	var result Flags[T]
	return result.Set(flags...)
}

// Unset clears the specified flags and returns the new value
func (f Flags[T]) Unset(flags ...T) Flags[T] {
	result := f.value
	for _, flag := range flags {
		result &= ^flag
	}
	return Flags[T]{value: result}
}

// Toggle toggles the specified flags and returns the new value
func (f Flags[T]) Toggle(flags ...T) Flags[T] {
	result := f.value
	for _, flag := range flags {
		result ^= flag
	}
	return Flags[T]{value: result}
}

func (f Flags[T]) Uint() T {
	return f.value
}

func (f Flags[T]) Ptr() *Flags[T] {
	return &f
}

func (f Flags[T]) String() string {
	return fmt.Sprintf("%d", f.value)
}

// MarshalJSON implements the json.Marshaler interface
func (f Flags[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.value)
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (f *Flags[T]) UnmarshalJSON(data []byte) error {
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	f.value = value
	return nil
}
