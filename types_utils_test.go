package jet

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func Test_ConvertFromMap(t *testing.T) {
	type Test struct {
		Name string `json:"name,omitempty"`
		Age  int    `json:"age,omitempty"`
		Sex  rune   `json:"sex,omitempty"`
	}

	map1 := map[string]interface{}{
		"name": "val1",
		"age":  12,
	}

	map2 := map[string]interface{}{
		"name": "val1",
		"age":  13.,
		"sex":  77,
	}

	res, err := ConvertFromMap[Test](map1)
	assert.NoError(t, err)
	assert.Equal(t, "val1", res.Name)
	assert.Equal(t, 12, res.Age)
	assert.Equal(t, int32(0), res.Sex)

	res, err = ConvertFromMap[Test](map2)
	assert.NoError(t, err)
	assert.Equal(t, "val1", res.Name)
	assert.Equal(t, 13, res.Age)
	assert.Equal(t, 'M', res.Sex)
}

func Test_ConvertToMap(t *testing.T) {
	type Test struct {
		Name string `json:"name,omitempty"`
		Age  int    `json:"age,omitempty"`
		Sex  rune   `json:"sex,omitempty"`
	}

	obj1 := Test{
		Name: "val1",
	}

	obj2 := Test{
		Name: "val1",
		Sex:  'M',
		Age:  12,
	}

	res, err := ConvertToMap(obj1)
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "val1", res["name"])

	res, err = ConvertToMap(obj2)
	assert.NoError(t, err)
	assert.Len(t, res, 3)
	assert.Equal(t, "val1", res["name"])
	assert.Equal(t, 77., res["sex"])
	assert.Equal(t, 12., res["age"])
}

func Test_MapsEqual(t *testing.T) {
	tests := []struct {
		name  string
		m1    map[string]interface{}
		m2    map[string]interface{}
		equal bool
	}{
		{
			name:  "Both nils",
			m1:    nil,
			m2:    nil,
			equal: true,
		},
		{
			name:  "Nil vs empty",
			m1:    make(map[string]interface{}),
			m2:    nil,
			equal: false,
		},
		{
			name: "contains second",
			m1: map[string]interface{}{
				"k": "v",
			},
			m2: map[string]interface{}{
				"k": "v",
				"v": "k",
			},
			equal: false,
		},
		{
			name: "contains first",
			m1: map[string]interface{}{
				"k": "v",
				"v": "k",
			},
			m2: map[string]interface{}{
				"k": "v",
			},
		},
		{
			name:  "Both nils",
			m1:    nil,
			m2:    nil,
			equal: true,
		},
		{
			name:  "Nil vs empty",
			m1:    make(map[string]interface{}),
			m2:    nil,
			equal: false,
		},
		{
			name: "Single values",
			m1: map[string]interface{}{
				"k": "v",
			},
			m2: map[string]interface{}{
				"k": "v",
			},
			equal: true,
		},
		{
			name: "Complex values",
			m1: map[string]interface{}{
				"k": struct {
					Value string
				}{
					Value: "value",
				},
			},
			m2: map[string]interface{}{
				"k": struct {
					Value string
				}{
					Value: "value",
				},
			},
			equal: true,
		},
		{
			name: "Multiple values",
			m1: map[string]interface{}{
				"k1": "v1",
				"k2": 100,
				"k3": true,
			},
			m2: map[string]interface{}{
				"k1": "v1",
				"k2": 100,
				"k3": true,
			},
			equal: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.equal, MapsEqual(tt.m1, tt.m2))
		})
	}
}

func Test_MapToLowerCamelKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:     "nil maps",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty maps",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
		{
			name:     "one level map",
			input:    map[string]interface{}{"Key": "value"},
			expected: map[string]interface{}{"key": "value"},
		},
		{
			name:     "multi words key",
			input:    map[string]interface{}{"VeryComplexKey": "value"},
			expected: map[string]interface{}{"veryComplexKey": "value"},
		},
		{
			name:     "multi level map",
			input:    map[string]interface{}{"Key": map[string]interface{}{"AnotherKey": "value", "SecondKey": map[string]interface{}{"Key": "value"}}},
			expected: map[string]interface{}{"key": map[string]interface{}{"anotherKey": "value", "secondKey": map[string]interface{}{"key": "value"}}},
		},
		{
			name:     "only keys",
			input:    map[string]interface{}{"Key": "VALUE"},
			expected: map[string]interface{}{"key": "VALUE"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, MapToLowerCamelKeys(tt.input))
		})
	}
}

func Test_MapInterfacesToBytesAndBack(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
	}{
		{
			name: "nil maps",
			m:    nil,
		}, {
			name: "empty maps",
			m:    map[string]interface{}{},
		}, {
			name: "map one value",
			m:    map[string]interface{}{"key1": "value1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytes := MapInterfacesToBytes(tt.m)
			m := BytesToMapInterfaces(bytes)
			assert.Equal(t, tt.m, m)
		})
	}
}

func Test_MapInterfacesToBytesNestedTypesAndBack(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
	}{
		{
			name: "diff type values",
			m: map[string]interface{}{
				"key1": "value1",
				"key2": float64(10),
				"key3": 98.2,
			},
		}, {
			name: "diff type values with map value",
			m: map[string]interface{}{
				"key1": "value1",
				"key2": float64(10),
				"key3": 98.2,
				"key4": map[string]interface{}{"key4internal1": float64(10), "key4internal2": "value2"}},
		}, {
			name: "diff type values with map value",
			m: map[string]interface{}{
				"key1": "value1",
				"key2": float64(10),
				"key3": 98.2,
				"key4": map[string]interface{}{
					"key4internal1": float64(10),
					"key4internal2": map[string]interface{}{
						"key4internal1": float64(10),
						"key4internal2": "value2"}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytes := MapInterfacesToBytes(tt.m)
			m := BytesToMapInterfaces(bytes)
			assertMap(t, tt.m, m)
		})
	}
}

func Test_StringsToInterfaces(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		expected []interface{}
	}{
		{
			name:     "nil slice",
			slice:    nil,
			expected: nil,
		}, {
			name:     "empty slice",
			slice:    []string{},
			expected: []interface{}{},
		}, {
			name:     "two value",
			slice:    []string{"value1", "value2"},
			expected: []interface{}{"value1", "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sl := StringsToInterfaces(tt.slice)
			assert.Equal(t, tt.expected, sl)
		})
	}

}

func assertMap(t *testing.T, expectedM, actualM map[string]interface{}) {
	assert.Equal(t, len(expectedM), len(actualM))
	for k, v := range expectedM {
		if internalV, ok := v.(map[string]interface{}); ok {
			assertMap(t, internalV, actualM[k].(map[string]interface{}))
		} else {
			assert.Equal(t, v, actualM[k])
		}
	}
}

func Test_ParseFloat32(t *testing.T) {
	assert.Nil(t, ParseFloat32(""))
	assert.Nil(t, ParseFloat32(" "))
	assert.Nil(t, ParseFloat32("qwrqwrqwr"))
	assert.Equal(t, float32(100.0), *ParseFloat32("100"))
	assert.Equal(t, float32(100.5), *ParseFloat32("100.5"))
	assert.Equal(t, float32(-100.5), *ParseFloat32("-100.5"))
}

func Test_Rounds(t *testing.T) {
	assert.Equal(t, 10.01, Round100(10.009))
	assert.Equal(t, 10., Round100(10.004))
	assert.Equal(t, 10., Round100(10.))
	assert.Equal(t, 3.35, Round100(3.3456))
	assert.Equal(t, 10.0001, Round10000(10.00009))
	assert.Equal(t, 10., Round10000(10.00004))
	assert.Equal(t, 10., Round10000(10.))
	assert.Equal(t, 3.8935, Round10000(3.893456))
}

func Test_Empty(t *testing.T) {
	assert.Equal(t, true, IsEmpty(""))
	assert.Equal(t, true, IsEmpty(nil))
	assert.Equal(t, true, IsEmpty(0))
	assert.Equal(t, true, IsEmpty(struct{}{}))
}

func TestPaginateSlice(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	tests := []struct {
		name     string
		items    []int
		paging   PagingRequest
		expected []int
	}{
		{
			name:     "Basic paging",
			items:    items,
			paging:   PagingRequest{Size: 3, Index: 2},
			expected: []int{4, 5, 6},
		},
		{
			name:     "Default page size and index",
			items:    items,
			paging:   PagingRequest{Size: 0, Index: 0}, // should default to Size=100 and Index=1
			expected: items,
		},
		{
			name:     "Empty slice",
			items:    []int{},
			paging:   PagingRequest{Size: 3, Index: 1},
			expected: []int{},
		},
		{
			name:     "Out-of-range index",
			items:    items,
			paging:   PagingRequest{Size: 3, Index: 5}, // should return empty as index is out of range
			expected: []int{},
		},
		{
			name:     "Index at end of slice",
			items:    items,
			paging:   PagingRequest{Size: 3, Index: 4}, // should return last element [10]
			expected: []int{10},
		},
		{
			name:     "Larger page size than items",
			items:    items,
			paging:   PagingRequest{Size: 15, Index: 1}, // should return all items since size is large
			expected: items,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PaginateSlice(tt.items, tt.paging)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestFlags_HasFlags tests the HasAllFlags method for Flags[uint16]
func TestFlags_HasFlags(t *testing.T) {
	tests := []struct {
		name       string
		flags      Flags[uint16]
		checkFlags []uint16
		want       bool
	}{
		{
			name:       "no flags set, checking no flags",
			flags:      NewFlags[uint16](),
			checkFlags: []uint16{},
			want:       true,
		},
		{
			name:       "no flags set, checking one flag",
			flags:      NewFlags[uint16](),
			checkFlags: []uint16{1},
			want:       false,
		},
		{
			name:       "one flag set, checking same flag",
			flags:      NewFlags[uint16](1),
			checkFlags: []uint16{1},
			want:       true,
		},
		{
			name:       "one flag set, checking different flag",
			flags:      NewFlags[uint16](1),
			checkFlags: []uint16{2},
			want:       false,
		},
		{
			name:       "multiple flags set, checking all of them",
			flags:      NewFlags[uint16](1, 2, 4), // 1 + 2 + 4
			checkFlags: []uint16{1, 2, 4},
			want:       true,
		},
		{
			name:       "multiple flags set, checking subset",
			flags:      NewFlags[uint16](1, 2, 4), // 1 + 2 + 4
			checkFlags: []uint16{1, 2},
			want:       true,
		},
		{
			name:       "multiple flags set, checking with one not set",
			flags:      NewFlags[uint16](1, 2, 4), // 1 + 2 + 4
			checkFlags: []uint16{1, 2, 8},
			want:       false,
		},
		{
			name:       "all possible flags set",
			flags:      NewFlags[uint16](1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768),
			checkFlags: []uint16{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.flags.HasAll(tt.checkFlags...)
			if got != tt.want {
				t.Errorf("HasFlags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlags_SetFlags(t *testing.T) {
	tests := []struct {
		name     string
		flags    Flags[uint16]
		setFlags []uint16
		want     uint16
	}{
		{
			name:     "set no flags on empty",
			flags:    NewFlags[uint16](),
			setFlags: []uint16{},
			want:     0,
		},
		{
			name:     "set one flag on empty",
			flags:    NewFlags[uint16](),
			setFlags: []uint16{1},
			want:     1,
		},
		{
			name:     "set multiple flags on empty",
			flags:    NewFlags[uint16](),
			setFlags: []uint16{1, 2, 4},
			want:     7, // 1 + 2 + 4
		},
		{
			name:     "set already set flag",
			flags:    NewFlags[uint16](1),
			setFlags: []uint16{1},
			want:     1,
		},
		{
			name:     "set mixed flags (some already set)",
			flags:    NewFlags[uint16](1, 4), // 1 + 4
			setFlags: []uint16{2, 4, 8},
			want:     15, // 1 + 2 + 4 + 8
		},
		{
			name:     "set all possible flags",
			flags:    NewFlags[uint16](),
			setFlags: []uint16{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768},
			want:     65535, // all 16 bits set
		},
		{
			name:     "set combination of complex flags",
			flags:    NewFlags[uint16](),
			setFlags: []uint16{0x0001, 0x0010, 0x0100, 0x1000},
			want:     0x1111, // binary 0001000100010001
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.flags.Set(tt.setFlags...)
			if got.value != tt.want {
				t.Errorf("SetFlags() = %v, want %v", got.value, tt.want)
			}
		})
	}
}

func TestFlags_UnSetFlags(t *testing.T) {
	tests := []struct {
		name       string
		flags      Flags[uint16]
		unsetFlags []uint16
		want       uint16
	}{
		{
			name:       "unset no flags on empty",
			flags:      NewFlags[uint16](),
			unsetFlags: []uint16{},
			want:       0,
		},
		{
			name:       "unset one flag on empty",
			flags:      NewFlags[uint16](),
			unsetFlags: []uint16{1},
			want:       0,
		},
		{
			name:       "unset one flag that is set",
			flags:      NewFlags[uint16](1),
			unsetFlags: []uint16{1},
			want:       0,
		},
		{
			name:       "unset multiple flags, all set",
			flags:      NewFlags[uint16](1, 2, 4), // 1 + 2 + 4
			unsetFlags: []uint16{1, 2, 4},
			want:       0,
		},
		{
			name:       "unset multiple flags, some set",
			flags:      NewFlags[uint16](1, 2, 4), // 1 + 2 + 4
			unsetFlags: []uint16{1, 2, 8},
			want:       4, // only 4 remains
		},
		{
			name:       "unset no existing flags",
			flags:      NewFlags[uint16](1, 2, 4), // 1 + 2 + 4
			unsetFlags: []uint16{8, 16},
			want:       7, // 1 + 2 + 4
		},
		{
			name:       "unset from all flags set",
			flags:      NewFlags[uint16](1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768), // all 16 bits set
			unsetFlags: []uint16{1, 4, 16, 64, 256, 1024, 4096, 16384},
			want:       43690, // binary 1010101010101010
		},
		{
			name:       "unset combination of complex flags",
			flags:      NewFlags[uint16](0x0001, 0x0010, 0x0100, 0x1000), // binary 0001000100010001
			unsetFlags: []uint16{0x0001, 0x0100},
			want:       0x1010, // binary 0001000000010000
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.flags.Unset(tt.unsetFlags...)
			if got.value != tt.want {
				t.Errorf("UnSetFlags() = %v, want %v", got.value, tt.want)
			}
		})
	}
}

func TestFlags_Combined(t *testing.T) {
	tests := []struct {
		name           string
		initialFlags   Flags[uint16]
		setFlags       []uint16
		unsetFlags     []uint16
		checkFlags     []uint16
		wantAfterSet   uint16
		wantAfterUnset uint16
		wantHasFlags   bool
	}{
		{
			name:           "combined operations test 1",
			initialFlags:   NewFlags[uint16](),
			setFlags:       []uint16{1, 2, 4},
			unsetFlags:     []uint16{2},
			checkFlags:     []uint16{1, 4},
			wantAfterSet:   7, // 1 + 2 + 4
			wantAfterUnset: 5, // 1 + 4
			wantHasFlags:   true,
		},
		{
			name:           "combined operations test 2",
			initialFlags:   NewFlags[uint16](2, 8), // 2 + 8
			setFlags:       []uint16{1, 4},
			unsetFlags:     []uint16{8},
			checkFlags:     []uint16{1, 2, 4},
			wantAfterSet:   15, // 1 + 2 + 4 + 8
			wantAfterUnset: 7,  // 1 + 2 + 4
			wantHasFlags:   true,
		},
		{
			name:           "combined operations test 3",
			initialFlags:   NewFlags[uint16](1, 2, 4, 8), // 1 + 2 + 4 + 8
			setFlags:       []uint16{16, 32},
			unsetFlags:     []uint16{2, 8, 16},
			checkFlags:     []uint16{1, 4, 32},
			wantAfterSet:   63, // 1 + 2 + 4 + 8 + 16 + 32
			wantAfterUnset: 37, // 1 + 4 + 32
			wantHasFlags:   true,
		},
		{
			name:           "combined operations with check failure",
			initialFlags:   NewFlags[uint16](1, 2, 4, 8), // 1 + 2 + 4 + 8
			setFlags:       []uint16{16},
			unsetFlags:     []uint16{2, 8},
			checkFlags:     []uint16{1, 2, 4},
			wantAfterSet:   31, // 1 + 2 + 4 + 8 + 16
			wantAfterUnset: 21, // 1 + 4 + 16
			wantHasFlags:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply set operation
			flags := tt.initialFlags
			got := flags.Set(tt.setFlags...)
			if got.value != tt.wantAfterSet {
				t.Errorf("SetFlags() = %v, want %v", got.value, tt.wantAfterSet)
			}

			// Apply unset operation on the result
			flags = got
			got = flags.Unset(tt.unsetFlags...)
			if got.value != tt.wantAfterUnset {
				t.Errorf("UnSetFlags() = %v, want %v", got.value, tt.wantAfterUnset)
			}

			// Check flags after all operations
			flags = got
			hasFlags := flags.HasAll(tt.checkFlags...)
			if hasFlags != tt.wantHasFlags {
				t.Errorf("HasFlags() = %v, want %v", hasFlags, tt.wantHasFlags)
			}
		})
	}
}
