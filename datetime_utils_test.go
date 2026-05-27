package jet

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_TimeParse(t *testing.T) {
	tm := HourMinTime{}
	tm, err := tm.Parse("10:00")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "10:00", tm.String())
	assert.Equal(t, 10, tm.Hour())
	assert.Equal(t, 0, tm.Minute())
}

func Test_TimeString(t *testing.T) {
	tm := HourMinTime{}
	tm, err := tm.Parse("10:00")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "10:00", tm.String())
}

func Test_Time_FromTime(t *testing.T) {
	assert.Equal(t, "15:37", HourMinTime{}.FromTime(time.Date(2022, 1, 1, 15, 37, 33, 12, time.UTC)).String())
	assert.Equal(t, "00:01", HourMinTime{}.FromTime(time.Date(2022, 1, 1, 0, 1, 0, 0, time.UTC)).String())
}

type testType struct {
	Time HourMinTime
}

func Test_Time_MarshalUnmarshal(t *testing.T) {
	tm := HourMinTime{}.MustParse("12:10")
	tp := &testType{Time: tm}
	b, err := json.Marshal(tp)
	assert.NoError(t, err)
	assert.NotEmpty(t, b)
	tp = &testType{}
	err = json.Unmarshal(b, &tp)
	assert.NoError(t, err)
	assert.NotEmpty(t, tp)
	assert.Equal(t, tp.Time, tm)
}

func Test_Date(t *testing.T) {
	tests := []struct {
		name   string
		input  time.Time
		output time.Time
	}{
		{
			name:   "Time to date",
			input:  time.Date(2022, 1, 1, 12, 23, 34, 122, time.UTC),
			output: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:   "Date to date",
			input:  time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			output: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.output, Date(tt.input))
		})
	}
}

func Test_TimeRange_ParseEmpty(t *testing.T) {
	tests := []struct {
		from string
		to   string
	}{
		{
			from: "",
			to:   "",
		},
		{
			from: "11:20",
			to:   "",
		},
		{
			from: "45:77",
			to:   "10:00",
		},
		{
			from: "_",
			to:   "12:23",
		},
	}
	for _, tt := range tests {
		t.Run(tt.from, func(t *testing.T) {
			actual := TimeRange{}.ParseOrEmpty(tt.from, tt.to)
			var expected *TimeRange
			assert.Equal(t, expected, actual)
		})
	}
}

func Test_TimeRange_Within(t *testing.T) {
	tests := []struct {
		name      string
		input     HourMinTime
		timeRange TimeRange
		res       bool
	}{
		{
			name:      "within",
			input:     HourMinTime{}.FromTime(time.Date(2022, 1, 1, 12, 23, 34, 122, time.UTC)),
			timeRange: TimeRange{}.MustParse("11:10", "13:15"),
			res:       true,
		},
		{
			name:      "not within",
			input:     HourMinTime{}.FromTime(time.Date(2022, 1, 1, 14, 23, 34, 122, time.UTC)),
			timeRange: TimeRange{}.MustParse("11:10", "13:15"),
			res:       false,
		},
		{
			name:      "within left edge",
			input:     HourMinTime{}.FromTime(time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)),
			timeRange: TimeRange{}.MustParse("00:00", "13:15"),
			res:       true,
		},
		{
			name:      "within right edge",
			input:     HourMinTime{}.FromTime(time.Date(2022, 1, 1, 13, 15, 0, 0, time.UTC)),
			timeRange: TimeRange{}.MustParse("00:00", "13:15"),
			res:       true,
		},
		{
			name:      "when over midnight",
			input:     HourMinTime{}.FromTime(time.Date(2022, 1, 1, 23, 15, 0, 0, time.UTC)),
			timeRange: TimeRange{}.MustParse("22:00", "07:00"),
			res:       true,
		},
		{
			name:      "when over midnight 2",
			input:     HourMinTime{}.FromTime(time.Date(2022, 1, 1, 06, 15, 0, 0, time.UTC)),
			timeRange: TimeRange{}.MustParse("22:00", "07:00"),
			res:       true,
		},
		{
			name:      "when over midnight not within",
			input:     HourMinTime{}.FromTime(time.Date(2022, 1, 1, 07, 15, 0, 0, time.UTC)),
			timeRange: TimeRange{}.MustParse("22:00", "07:00"),
			res:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.res, tt.timeRange.Within(tt.input))
		})
	}
}

func Test_TimeZone(t *testing.T) {
	date := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, "2023-01-01 06:00:00", date.In(GetTzLocation(TzP6)).Format("2006-01-02 15:04:05"))
	actual, err := ToTz(date, TzP6)
	assert.NoError(t, err)
	assert.Equal(t, "2023-01-01 06:00:00", actual.Format("2006-01-02 15:04:05"))
}

func Test_GenerateTimeSeries(t *testing.T) {
	tests := []struct {
		name   string
		from   time.Time
		to     time.Time
		period time.Duration
		res    []time.Time
	}{
		{
			name:   "2 mins",
			from:   time.Date(2023, 1, 1, 1, 1, 0, 0, time.UTC),
			to:     time.Date(2023, 1, 1, 1, 3, 0, 0, time.UTC),
			period: time.Minute,
			res: []time.Time{
				time.Date(2023, 1, 1, 1, 1, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 1, 2, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 1, 3, 0, 0, time.UTC),
			},
		},
		{
			name:   "to & from same",
			from:   time.Date(2023, 1, 1, 1, 1, 0, 0, time.UTC),
			to:     time.Date(2023, 1, 1, 1, 1, 0, 0, time.UTC),
			period: time.Minute,
			res: []time.Time{
				time.Date(2023, 1, 1, 1, 1, 0, 0, time.UTC),
			},
		},
		{
			name:   "to after from",
			from:   time.Date(2023, 1, 1, 1, 2, 0, 0, time.UTC),
			to:     time.Date(2023, 1, 1, 1, 1, 0, 0, time.UTC),
			period: time.Minute,
			res: []time.Time{
				time.Date(2023, 1, 1, 1, 2, 0, 0, time.UTC),
			},
		},
		{
			name:   "2 hour",
			from:   time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC),
			to:     time.Date(2023, 1, 1, 5, 0, 0, 0, time.UTC),
			period: 2 * time.Hour,
			res: []time.Time{
				time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 3, 0, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 5, 0, 0, 0, time.UTC),
			},
		},
		{
			name:   "2 hour",
			from:   time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC),
			to:     time.Date(2023, 1, 1, 6, 0, 0, 0, time.UTC),
			period: 2 * time.Hour,
			res: []time.Time{
				time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 3, 0, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 5, 0, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 6, 0, 0, 0, time.UTC),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.res, GenerateTimeSeries(tt.from, tt.to, tt.period))
		})
	}
}

func Test_MinTime(t *testing.T) {
	tests := []struct {
		name string
		in   []time.Time
		res  *time.Time
	}{
		{
			name: "nil",
			in:   nil,
			res:  nil,
		},
		{
			name: "empty",
			in:   []time.Time{},
			res:  nil,
		},
		{
			name: "single",
			in:   []time.Time{time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC)},
			res:  TimePtr(time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC)),
		},
		{
			name: "multiple",
			in:   []time.Time{time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC)},
			res:  TimePtr(time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC)),
		},
		{
			name: "equal",
			in:   []time.Time{time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC)},
			res:  TimePtr(time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.res, MinTime(tt.in...))
		})
	}
}

func Test_MaxTime(t *testing.T) {
	tests := []struct {
		name string
		in   []time.Time
		res  *time.Time
	}{
		{
			name: "nil",
			in:   nil,
			res:  nil,
		},
		{
			name: "empty",
			in:   []time.Time{},
			res:  nil,
		},
		{
			name: "single",
			in:   []time.Time{time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC)},
			res:  TimePtr(time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC)),
		},
		{
			name: "multiple",
			in:   []time.Time{time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC)},
			res:  TimePtr(time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC)),
		},
		{
			name: "equal",
			in:   []time.Time{time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC)},
			res:  TimePtr(time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.res, MaxTime(tt.in...))
		})
	}
}

func Test_IsTimeZoneIANA(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  bool
	}{
		{name: "valid", in: "Europe/Beograd", out: true},
		{name: "valid UTC", in: "UTC", out: true},
		{name: "not valid", in: "Europe", out: false},
		{name: "empty", in: "", out: false},
		{name: "not valid", in: "Europe/Beograd/Beograd", out: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.out, IsTimeZoneIANA(tt.in))
		})
	}
}
