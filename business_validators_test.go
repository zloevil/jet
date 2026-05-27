package jet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_IsRussianPhoneValid(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"79991234567", true},
		{"89991234567", true},
		{"7999123456", false},   // only 9 digits after prefix
		{"799912345678", false}, // too long
		{"19991234567", false},  // wrong prefix
		{"+79991234567", false}, // plus not allowed
		{"", false},
	}
	for _, tt := range tests {
		assert.Equalf(t, tt.want, IsRussianPhoneValid(tt.in), "input %q", tt.in)
	}
}

func Test_IsCoordinateValid(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"55.123456", true},
		{"-12.1234567", true},
		{"1.12345", true},
		{"555.123456", false},  // 3 integer digits
		{"55.1234", false},     // too few fractional digits
		{"55.12345678", false}, // too many fractional digits
		{"55", false},          // no fraction
		{"abc", false},
		{"", false},
	}
	for _, tt := range tests {
		assert.Equalf(t, tt.want, IsCoordinateValid(tt.in), "input %q", tt.in)
	}
}
