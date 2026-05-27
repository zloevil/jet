package http

import (
	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
	"testing"
)

var logger = jet.InitLogger(&jet.LogConfig{Level: jet.InfoLevel})
var logf = func() jet.CLogger {
	return jet.L(logger)
}

type sortConvertTestSuite struct {
	jet.Suite
}

func (s *sortConvertTestSuite) SetupSuite() {
	s.Suite.Init(logf)
}

func TestTagSuite(t *testing.T) {
	suite.Run(t, new(sortConvertTestSuite))
}

func (s *sortConvertTestSuite) Test_ParseSortBy() {
	tests := []struct {
		name       string
		sortString string
		want       []*jet.SortRequest
		wantErr    bool
	}{
		{
			name:       "Empty string",
			sortString: "",
			want:       nil,
		},
		{
			name:       "real example",
			sortString: "reportedAt desc",
			want: []*jet.SortRequest{
				{
					Field: "reportedAt",
					Desc:  true,
				},
			},
		},
		{
			name:       "All ok (without missings)",
			sortString: "field1,field2 desc",
			want: []*jet.SortRequest{
				{
					Field: "field1",
					Desc:  false,
				},
				{
					Field: "field2",
					Desc:  true,
				},
			},
		},
		{
			name:       "All ok (with missings)",
			sortString: "field1 asc first,field2 desc last,field3 asc",
			want: []*jet.SortRequest{
				{
					Field:     "field1",
					Desc:      false,
					NullsLast: false,
				},
				{
					Field:     "field2",
					Desc:      true,
					NullsLast: true,
				},
				{
					Field: "field3",
					Desc:  false,
				},
			},
		},
		{
			name:       "Whitespaces",
			sortString: " field1    asc  , field2 desc  ",
			wantErr:    true,
		},
		{
			name:       "1 field",
			sortString: "field1 asc",
			want: []*jet.SortRequest{
				{
					Field: "field1",
					Desc:  false,
				},
			},
		},
		{
			name:       "1 field only name",
			sortString: "field1",
			want: []*jet.SortRequest{
				{
					Field: "field1",
					Desc:  false,
				},
			},
		},
		{
			name:       "Illegal sort mode",
			sortString: "field1 asc,field2 illegal_mode",
			wantErr:    true,
		},
		{
			name:       "Illegal missing mode",
			sortString: "field1 asc,field2 desc illegal_mode",
			wantErr:    true,
		},
		{
			name:       "Illegal syntax 1",
			sortString: "field1 asc,field2=desc",
			wantErr:    true,
		},
		{
			name:       "Illegal syntax 2",
			sortString: "field1 asc,desc=field2",
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		res, err := ParseSortBy(s.Ctx, tt.sortString)
		s.Equal(tt.want, res)
		s.Equal(tt.wantErr, err != nil)
	}
}
