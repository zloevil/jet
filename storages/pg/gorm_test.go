package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_StringToNull(t *testing.T) {
	assert.Nil(t, StringToNull(""), "empty string must become nil")

	got := StringToNull("x")
	require.NotNil(t, got)
	assert.Equal(t, "x", *got)
}

func Test_NullToString(t *testing.T) {
	assert.Equal(t, "", NullToString(nil), "nil must become empty string")

	v := "x"
	assert.Equal(t, "x", NullToString(&v))
}

func Test_PagingLimit(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{"zero -> default", 0, PageSizeDefault},
		{"negative -> default", -10, PageSizeDefault},
		{"one -> one", 1, 1},
		{"under max -> passthrough", 50, 50},
		{"at max -> max", PageSizeMaxLimit, PageSizeMaxLimit},
		{"over max -> clamped to max", PageSizeMaxLimit + 1, PageSizeMaxLimit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, PagingLimit(tt.in))
		})
	}
}

func Test_GetEmptyJson(t *testing.T) {
	j, err := GetEmptyJson()
	require.NoError(t, err)
	require.NotNil(t, j)
	assert.Equal(t, "{}", string(*j))
}

func Test_MapToJsonb(t *testing.T) {
	t.Run("nil map -> empty json", func(t *testing.T) {
		j, err := MapToJsonb[string, int](nil)
		require.NoError(t, err)
		require.NotNil(t, j)
		assert.Equal(t, "{}", string(*j))
	})

	t.Run("populated map -> jsonb", func(t *testing.T) {
		j, err := MapToJsonb(map[string]int{"a": 1, "b": 2})
		require.NoError(t, err)
		require.NotNil(t, j)
		assert.JSONEq(t, `{"a":1,"b":2}`, string(*j))
	})
}

type jsonbSample struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func Test_ToJsonb_FromJsonb_RoundTrip(t *testing.T) {
	t.Run("nil payload -> empty json", func(t *testing.T) {
		j, err := ToJsonb[jsonbSample](nil)
		require.NoError(t, err)
		require.NotNil(t, j)
		assert.Equal(t, "{}", string(*j))
	})

	t.Run("round trip preserves value", func(t *testing.T) {
		in := &jsonbSample{Name: "alice", Age: 30}

		j, err := ToJsonb(in)
		require.NoError(t, err)
		require.NotNil(t, j)
		assert.JSONEq(t, `{"name":"alice","age":30}`, string(*j))

		out, err := FromJsonb[jsonbSample](j)
		require.NoError(t, err)
		require.NotNil(t, out)
		assert.Equal(t, *in, *out)
	})
}

func Test_FromJsonb_Nil(t *testing.T) {
	out, err := FromJsonb[jsonbSample](nil)
	require.NoError(t, err)
	assert.Nil(t, out)
}
