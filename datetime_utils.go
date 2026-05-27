package jet

import (
	"errors"
	"fmt"
	"github.com/araddon/dateparse"
	"regexp"
	"time"
)

const (
	ErrCodeNotSupportedTz = "DTU-001"
)

var (
	ErrNotSupportedTz = func(err error, tz string) error {
		return NewAppErrBuilder(ErrCodeNotSupportedTz, "timezone isn't supported").Wrap(err).F(KV{"tz": tz}).Business().Err()
	}
)

func MillisFromTime(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

func TimeFromMillis(millis int64) time.Time {
	return time.Unix(0, millis*int64(time.Millisecond))
}

var timeLayout = "15:04"

// HourMinTime - time (hour:min) representation in format 15:04
type HourMinTime struct {
	tm time.Time
}

// TimeRange represents time interval in format [15:00, 18:00]
type TimeRange [2]HourMinTime

func (t *HourMinTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.tm.Format(timeLayout) + `"`), nil
}

func (t *HourMinTime) UnmarshalJSON(b []byte) error {
	s := string(b)
	// len(`"23:59"`) == 7
	if len(s) != 7 {
		return errors.New("time parsing error")
	}
	ret, err := time.Parse(timeLayout, s[1:6])
	if err != nil {
		return err
	}
	t.tm = ret
	return nil
}

func (t HourMinTime) Parse(s string) (HourMinTime, error) {
	tm, err := time.Parse(timeLayout, s)
	if err != nil {
		return HourMinTime{}, err
	}
	t.tm = tm
	return t, nil
}

func (t HourMinTime) MustParse(s string) HourMinTime {
	ret, _ := t.Parse(s)
	return ret
}

func (t HourMinTime) FromTime(tm time.Time) HourMinTime {
	modified, _ := t.Parse(fmt.Sprintf("%02d:%02d", tm.Hour(), tm.Minute()))
	return modified
}

func (t HourMinTime) Before(other HourMinTime) bool {
	return t.tm.Before(other.tm)
}

func (t HourMinTime) Unix() int64 {
	return t.tm.Unix()
}

func (t HourMinTime) String() string {
	return t.tm.Format(timeLayout)
}

func (t HourMinTime) Hour() int {
	return t.tm.Hour()
}

func (t HourMinTime) Minute() int {
	return t.tm.Minute()
}

func (t TimeRange) ParseOrEmpty(from, to string) *TimeRange {
	tr, _ := TimeRange{}.Parse(from, to)
	return tr
}

func (t TimeRange) MustParse(from, to string) TimeRange {
	tr, err := TimeRange{}.Parse(from, to)
	if err != nil {
		panic(err)
	}
	return *tr
}

func (t TimeRange) Parse(from, to string) (*TimeRange, error) {
	tFrom, err := time.Parse(timeLayout, from)
	if err != nil {
		return nil, err
	}
	tTo, err := time.Parse(timeLayout, to)
	if err != nil {
		return nil, err
	}
	return &TimeRange{HourMinTime{}.FromTime(tFrom), HourMinTime{}.FromTime(tTo)}, nil
}

func (t TimeRange) Valid() bool {
	return t[0] != t[1]
}

func (t TimeRange) ValidRange() bool {
	return t[0].Before(t[1])
}

func (t TimeRange) Within(tm HourMinTime) bool {
	start, end := t[0], t[1]
	if end.Before(start) {
		if tm.Before(start) {
			tm = HourMinTime{tm.tm.Add(time.Hour * 24)}
		}
		end = HourMinTime{t[1].tm.Add(time.Hour * 24)}
	}
	return start.Unix() <= tm.Unix() && end.Unix() >= tm.Unix()
}

// WithinExcl with left inclusive and right exclusive
func (t TimeRange) WithinExcl(tm HourMinTime) bool {
	start, end := t[0], t[1]
	if end.Before(start) {
		if tm.Before(start) {
			tm = HourMinTime{tm.tm.Add(time.Hour * 24)}
		}
		end = HourMinTime{t[1].tm.Add(time.Hour * 24)}
	}
	return start.Unix() <= tm.Unix() && end.Unix() > tm.Unix()
}

func (t *TimeRange) StartTime() string {
	if t == nil {
		return ""
	}
	return t[0].String()
}

func (t *TimeRange) EndTime() string {
	if t == nil {
		return ""
	}
	return t[1].String()
}

// Now is the current time
func Now() time.Time {
	return time.Now().Round(time.Microsecond).UTC()
}

// NowNanos is the current time in UNIX NANO format
func NowNanos() int64 {
	return time.Now().UTC().UnixNano()
}

// NowMillis is the current time in millis
func NowMillis() int64 {
	return Millis(time.Now().UTC())
}

// Millis is a convenience method to get milliseconds since epoch for provided HourMinTime.
func Millis(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

// Diff properly calculates difference between two dates in year, month etc.
func Diff(a, b time.Time) (year, month, day, hour, min, sec int) {
	if a.Location() != b.Location() {
		b = b.In(a.Location())
	}
	if a.After(b) {
		a, b = b, a
	}
	y1, M1, d1 := a.Date()
	y2, M2, d2 := b.Date()

	h1, m1, s1 := a.Clock()
	h2, m2, s2 := b.Clock()

	year = int(y2 - y1)
	month = int(M2 - M1)
	day = int(d2 - d1)
	hour = int(h2 - h1)
	min = int(m2 - m1)
	sec = int(s2 - s1)

	// Normalize negative values
	if sec < 0 {
		sec += 60
		min--
	}
	if min < 0 {
		min += 60
		hour--
	}
	if hour < 0 {
		hour += 24
		day--
	}
	if day < 0 {
		// days in month:
		t := time.Date(y1, M1, 32, 0, 0, 0, 0, time.UTC)
		day += 32 - t.Day()
		month--
	}
	if month < 0 {
		month += 12
		year--
	}
	return
}

func ToStringDate(date *time.Time) string {
	if date == nil {
		return ""
	}
	return date.Format(time.RFC3339)
}

// Date returns a date of a passed timestamp without time
func Date(date time.Time) time.Time {
	y, m, d := date.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// NowDate returns current date without time
func NowDate() time.Time {
	return Date(Now())
}

// ParseDateAny parses multiple formats of dates
func ParseDateAny(s string) *time.Time {
	if s == "" {
		return nil
	}
	if d, err := dateparse.ParseAny(s); err == nil {
		return &d
	}
	return nil
}

// DayOfWeek specifies days of week
type DayOfWeek string

const (
	Monday    DayOfWeek = "Mon"
	Tuesday   DayOfWeek = "Tue"
	Wednesday DayOfWeek = "Wed"
	Thursday  DayOfWeek = "Thu"
	Friday    DayOfWeek = "Fri"
	Saturday  DayOfWeek = "Sat"
	Sunday    DayOfWeek = "Sun"
)

func (d *DayOfWeek) IsValid(s string) bool {
	return s == string(Monday) || s == string(Tuesday) || s == string(Wednesday) || s == string(Thursday) ||
		s == string(Friday) || s == string(Saturday) || s == string(Sunday)
}

type DaysOfWeek map[DayOfWeek]struct{}

func (d DaysOfWeek) IsValid() bool {
	for k := range d {
		if !k.IsValid(string(k)) {
			return false
		}
	}
	return true
}

func Overlapped(startA, endA, startB, endB time.Time) bool {
	return startA.Before(endB) && startB.Before(endA)
}

const (
	TzUTC  = "UTC"
	TzP13  = "UTC+13"
	TzP12  = "UTC+12"
	TzP11  = "UTC+11"
	TzP10  = "UTC+10"
	TzP9   = "UTC+9"
	TzP8   = "UTC+8"
	TzP7   = "UTC+7"
	TzP6p5 = "UTC+6:30"
	TzP6   = "UTC+6"
	TzP5   = "UTC+5"
	TzP4   = "UTC+4"
	TzP3   = "UTC+3"
	TzP2   = "UTC+2"
	TzP1   = "UTC+1"
	TzM1   = "UTC-1"
	TzM2   = "UTC-2"
	TzM3   = "UTC-3"
	TzM4   = "UTC-4"
	TzM5   = "UTC-5"
	TzM6   = "UTC-6"
	TzM7   = "UTC-7"
	TzM8   = "UTC-8"
	TzM9   = "UTC-9"
	TzM10  = "UTC-10"
	TzM11  = "UTC-11"
)

var (
	tzOffsets = map[string]time.Duration{
		TzUTC:  0,
		TzP13:  13 * time.Hour,
		TzP12:  12 * time.Hour,
		TzP11:  11 * time.Hour,
		TzP10:  10 * time.Hour,
		TzP9:   9 * time.Hour,
		TzP8:   8 * time.Hour,
		TzP7:   7 * time.Hour,
		TzP6p5: 6*time.Hour + 30*time.Minute,
		TzP6:   6 * time.Hour,
		TzP5:   5 * time.Hour,
		TzP4:   4 * time.Hour,
		TzP3:   3 * time.Hour,
		TzP2:   2 * time.Hour,
		TzP1:   time.Hour,
		TzM1:   -time.Hour,
		TzM2:   -2 * time.Hour,
		TzM3:   -3 * time.Hour,
		TzM4:   -4 * time.Hour,
		TzM5:   -5 * time.Hour,
		TzM6:   -6 * time.Hour,
		TzM7:   -7 * time.Hour,
		TzM8:   -8 * time.Hour,
		TzM9:   -9 * time.Hour,
		TzM10:  -10 * time.Hour,
		TzM11:  -11 * time.Hour,
	}
	tzLocations = map[string]*time.Location{}
)

func TzValid(tz string) bool {
	return tzLocations[tz] != nil
}

func GetTzLocation(tz string) *time.Location {
	return tzLocations[tz]
}

func init() {
	for k, v := range tzOffsets {
		tzLocations[k] = time.FixedZone(k, int(v.Seconds()))
	}
}

func ToTz(t time.Time, tz string) (time.Time, error) {
	if tz == "" {
		return t, nil
	}
	// first check predefined locations
	loc := GetTzLocation(tz)
	if loc == nil {
		// try to load timezone
		var err error
		loc, err = time.LoadLocation(tz)
		if err != nil {
			return t, ErrNotSupportedTz(err, tz)
		}
	}
	//set timezone,
	return t.In(loc), nil
}

// GenerateTimeSeries takes period of time and generates slice of timestamps with the given period step
func GenerateTimeSeries(from, to time.Time, period time.Duration) []time.Time {
	r := []time.Time{from}
	if from == to {
		return r
	}
	cur := from
	for to.Sub(cur) > period {
		cur = cur.Add(period)
		r = append(r, cur)
	}
	if to.After(cur) {
		r = append(r, to)
	}
	return r
}

// MinTime returns minimum time of the provided slice
func MinTime(times ...time.Time) *time.Time {
	if len(times) == 0 {
		return nil
	}
	r := times[0]
	for _, t := range times {
		if t.Before(r) {
			r = t
		}
	}
	return &r
}

// MaxTime returns maximum time of the provided slice
func MaxTime(times ...time.Time) *time.Time {
	if len(times) == 0 {
		return nil
	}
	r := times[0]
	for _, t := range times {
		if t.After(r) {
			r = t
		}
	}
	return &r
}

// IsTimeZoneIANA returns true if provided valid IANA time zone
func IsTimeZoneIANA(tz string) bool {
	if tz == TzUTC {
		return true
	}
	// TODO: it's simplification as it has to be validated against the real IANA database
	ok, _ := regexp.MatchString(`^[A-Za-z]{3,20}/[A-Za-z]{3,20}$`, tz)
	return ok
}

// TimePeriod represents period of time
type TimePeriod struct {
	From *time.Time
	To   *time.Time
}

// Valid if time period is valid
func (t TimePeriod) Valid() bool {
	return (t.From == nil && t.To == nil) ||
		(t.From != nil && t.To != nil && !(t.From.After(*t.To)))
}
