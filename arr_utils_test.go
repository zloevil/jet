package jet

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func Test_SplitArrByLen(t *testing.T) {
	tests := []struct {
		name string
		data []string
		max  int
		want [][]string
	}{
		{
			name: "Empty slice",
			data: nil,
			max:  5,
			want: nil,
		},
		{
			name: "Elements fit perfectly",
			data: []string{"hi", "go", "yes"},
			max:  6,
			want: [][]string{
				{"hi", "go"}, {"yes"},
			},
		},
		{
			name: "Empty elements",
			data: []string{"", "", "", "", "", "", "", "", ""},
			max:  6,
			want: [][]string{
				{"", "", "", "", "", "", "", "", ""},
			},
		},
		{
			name: "Splitting required",
			data: []string{"hello", "world", "golang", "is", "awesome"},
			max:  10,
			want: [][]string{
				{"hello", "world"},
				{"golang", "is"},
				{"awesome"},
			},
		},
		{
			name: "Mix of short and long words",
			data: []string{"a", "bc", "def", "ghij", "klmno", "pqrstuv"},
			max:  8,
			want: [][]string{
				{"a", "bc", "def"},
				{"ghij"}, {"klmno"},
				{"pqrstuv"},
			},
		},
		{
			name: "All max",
			data: []string{"a", "b", "c", "d", "e"},
			max:  1,
			want: [][]string{
				{"a"}, {"b"}, {"c"}, {"d"}, {"e"},
			},
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			rs, err := SplitArrByItemLen(tt.data, tt.max)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, rs)
		})
	}
}

func Test_SplitArr(t *testing.T) {
	tests := []struct {
		arr  []int
		size int
		res  [][]int
	}{
		{
			arr:  []int{1, 2, 3},
			size: 45,
			res:  [][]int{{1, 2, 3}},
		},
		{
			arr:  []int{1, 2, 3},
			size: 1,
			res:  [][]int{{1}, {2}, {3}},
		},
		{
			arr:  []int{1, 2, 3},
			size: 2,
			res:  [][]int{{1, 2}, {3}},
		},
		{
			arr:  []int{1, 2, 3},
			size: 3,
			res:  [][]int{{1, 2, 3}},
		},
		{
			arr:  []int{},
			size: 45,
			res:  nil,
		},
		{
			arr:  nil,
			size: 45,
			res:  nil,
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.res, SplitArr(tt.arr, tt.size))
		})
	}
}
