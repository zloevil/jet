package jet

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"math"
	"math/rand"
	"reflect"
	"strconv"
	"time"
)

type Suite struct {
	suite.Suite
	logger    CLoggerFunc
	Ctx       context.Context
	suiteName string
}

func (s *Suite) Init(logger CLoggerFunc) {
	s.logger = logger
	if logger == nil {
		s.DefaultLogger()
	}
	s.resetContext()
}

func (s *Suite) DefaultLogger() {
	s.logger = func() CLogger {
		return L(InitLogger(&LogConfig{Level: TraceLevel})).Srv("test")
	}
}

func (s *Suite) L() CLogger {
	return s.logger().Cmp(s.suiteName).Mth(s.T().Name()).C(s.Ctx)
}

func (s *Suite) BeforeTest(suiteName, testName string) {
	s.resetContext()
	s.suiteName = suiteName
	s.logger().Cmp(suiteName).Mth(testName).C(s.Ctx).Inf("start")
}

func (s *Suite) AfterTest(suiteName, testName string) {
	l := s.logger().Cmp(suiteName).Mth(testName).C(s.Ctx)
	if s.T().Failed() {
		l.F(KV{"status": "fail"}).Inf("fail")
	} else {
		l.F(KV{"status": "ok"}).Inf("ok")
	}
}

func (s *Suite) resetContext() {
	s.Ctx = NewRequestCtx().WithNewRequestId().EN().TestApp().ToContext(context.Background())
}

func (s *Suite) f(v interface{}) string {
	return fmt.Sprintf("%T(%#v)", v, v)
}

func (s *Suite) fail(fields KV, errorMsg string, msgAndArgs ...interface{}) {
	l := s.logger().Mth(s.T().Name()).C(s.Ctx).St()
	for i, field := range fields {
		fields[i] = s.f(field)
	}
	if len(msgAndArgs) > 0 {
		fields["args"] = msgAndArgs
	}
	if len(fields) > 0 {
		l.F(fields)
	}
	l.Err(errorMsg)
	s.T().FailNow()
}

func (s *Suite) AssertAppErr(err error, code string) {
	s.Error(err)
	appEr, ok := IsAppErr(err)
	s.True(ok)
	s.Equal(code, appEr.Code())
}

func (s *Suite) Fatal(msgAndArgs ...interface{}) {
	s.logger().Mth(s.T().Name()).C(s.Ctx).St().Err(msgAndArgs...)
	s.T().Fatal(msgAndArgs...)
}

// Equal asserts that two objects are equal.
//
//	s.Equal(t, 123, 123)
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses). Function equality
// cannot be determined and will always fail.
func (s *Suite) Equal(expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	if ok := assert.Equal(s.T(), expected, actual); !ok {
		s.fail(KV{"expected": expected, "actual": actual}, "not equal", msgAndArgs...)
	}
}

// EqualValues asserts that two objects are equal or convertable to the same types
// and equal.
//
//	s.EqualValues(t, uint32(123), int32(123))
func (s *Suite) EqualValues(expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	if ok := assert.EqualValues(s.T(), expected, actual); !ok {
		s.fail(KV{"expected": expected, "actual": actual}, "not equal values", msgAndArgs...)
	}
}

// NotEqual asserts that the specified values are NOT equal.
//
//	s.NotEqual(t, obj1, obj2)
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses).
func (s *Suite) NotEqual(expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	if ok := assert.NotEqual(s.T(), expected, actual); !ok {
		s.fail(KV{"expected": expected, "actual": actual}, "equal", msgAndArgs...)
	}
}

// NotEqualValues asserts that two objects are not equal even when converted to the same type
//
//	s.NotEqualValues(t, obj1, obj2)
func (s *Suite) NotEqualValues(expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	if ok := assert.EqualValues(s.T(), expected, actual); !ok {
		s.fail(KV{"expected": expected, "actual": actual}, "equal values", msgAndArgs...)
	}
}

// Exactly asserts that two objects are equal in value and type.
//
//	s.Exactly(t, int32(123), int64(123))
func (s *Suite) Exactly(expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	if ok := assert.Exactly(s.T(), expected, actual); !ok {
		s.fail(KV{"expected": expected, "actual": actual}, "not exactly", msgAndArgs...)
	}
}

// Empty asserts that the specified object is empty.  I.e. nil, "", false, 0 or either
// a slice or a channel with len == 0.
//
//	s.Empty(t, obj)
func (s *Suite) Empty(object interface{}, msgAndArgs ...interface{}) {
	if ok := assert.Empty(s.T(), object); !ok {
		s.fail(KV{"object": object}, "not empty", msgAndArgs...)
	}
}

// NotEmpty asserts that the specified object is NOT empty.  I.e. not nil, "", false, 0 or either
// a slice or a channel with len == 0.
//
//	if s.NotEmpty(t, obj) {
//	  s.Equal(t, "two", obj[1])
//	}
func (s *Suite) NotEmpty(object interface{}, msgAndArgs ...interface{}) {
	if ok := assert.NotEmpty(s.T(), object); !ok {
		s.fail(KV{"object": object}, "empty", msgAndArgs...)
	}
}

// Exactly asserts that two objects are equal in value and type.
//
//	s.Exactly(t, int32(123), int64(123))
func (s *Suite) Nil(object interface{}, msgAndArgs ...interface{}) {
	if ok := assert.Nil(s.T(), object); !ok {
		s.fail(KV{"object": object}, "not nil", msgAndArgs...)
	}
}

// NotNil asserts that the specified object is not nil.
//
//	s.NotNil(t, err)
func (s *Suite) NotNil(object interface{}, msgAndArgs ...interface{}) {
	if ok := assert.NotNil(s.T(), object); !ok {
		s.fail(KV{"object": object}, "nil", msgAndArgs...)
	}
}

// getLen try to get length of object.
// return (false, 0) if impossible.
func getLen(x interface{}) (ok bool, length int) {
	v := reflect.ValueOf(x)
	defer func() {
		if e := recover(); e != nil {
			ok = false
		}
	}()
	return true, v.Len()
}

// Len asserts that the specified object has specific length.
// Len also fails if the object has a type that len() not accept.
//
//	s.Len(t, mySlice, 3)
func (s *Suite) Len(object interface{}, length int, msgAndArgs ...interface{}) {
	ok, l := getLen(object)
	if !ok {
		s.FailNow(fmt.Sprintf("\"%s\" could not be applied builtin len()", object), msgAndArgs...)
	}

	if l != length {
		s.fail(KV{"expected len": length, "got item(s)": l}, "not equal length", msgAndArgs...)
	}
}

// Implements asserts that an object is implemented by the specified interface.
//
//	s.Implements(t, (*MyInterface)(nil), new(MyObject))
func (s *Suite) Implements(interfaceObject interface{}, object interface{}, msgAndArgs ...interface{}) {
	if ok := assert.Implements(s.T(), interfaceObject, object); !ok {
		interfaceType := reflect.TypeOf(interfaceObject).Elem()
		s.fail(KV{"type": interfaceType, "object": object}, "not implements", msgAndArgs...)
	}
}

// IsType asserts that the specified objects are of the same type.
func (s *Suite) IsType(expectedType interface{}, object interface{}, msgAndArgs ...interface{}) {
	if ok := assert.IsType(s.T(), expectedType, object); !ok {
		s.fail(KV{"type": reflect.TypeOf(object), "object": reflect.TypeOf(expectedType)}, "not an equal types", msgAndArgs...)
	}
}

// Same asserts that two pointers reference the same object.
//
//	s.Same(t, ptr1, ptr2)
//
// Both arguments must be pointer variables. Pointer variable sameness is
// determined based on the equality of both type and value.
func (s *Suite) Same(expected, actual interface{}, msgAndArgs ...interface{}) {
	if ok := assert.Same(s.T(), expected, actual); !ok {
		s.fail(KV{"expected": fmt.Sprintf("%p %#v", expected, expected), "actual": fmt.Sprintf("%p %#v", actual, actual)}, "not the same links", msgAndArgs...)
	}
}

// NotSame asserts that two pointers do not reference the same object.
//
//	s.NotSame(t, ptr1, ptr2)
//
// Both arguments must be pointer variables. Pointer variable sameness is
// determined based on the equality of both type and value.
func (s *Suite) NotSame(expected, actual interface{}, msgAndArgs ...interface{}) {
	if ok := assert.NotSame(s.T(), expected, actual); !ok {
		s.fail(KV{"expected": fmt.Sprintf("%p %#v", expected, expected), "actual": fmt.Sprintf("%p %#v", actual, actual)}, "same links", msgAndArgs...)
	}
}

// True asserts that the specified value is true.
//
//	s.True(t, myBool)
func (s *Suite) True(value bool, msgAndArgs ...interface{}) {
	if !value {
		s.fail(KV{"value": value}, "should be true", msgAndArgs...)
	}
}

// False asserts that the specified value is false.
//
//	s.False(t, myBool)
func (s *Suite) False(value bool, msgAndArgs ...interface{}) {
	if value {
		s.fail(KV{"value": value}, "should be false", msgAndArgs...)
	}
}

// Contains asserts that the specified string, list(array, slice...) or map contains the
// specified substring or element.
//
//	s.Contains(t, "Hello World", "World")
//	s.Contains(t, ["Hello", "World"], "World")
//	s.Contains(t, {"Hello": "World"}, "Hello")
func (s *Suite) Contains(el, contains interface{}, msgAndArgs ...interface{}) {
	if ok := assert.Contains(s.T(), el, contains); !ok {
		s.fail(KV{"list": s.f(contains), "item": s.f(el)}, "not contains", msgAndArgs...)
	}
}

// NotContains asserts that the specified string, list(array, slice...) or map does NOT contain the
// specified substring or element.
//
//	s.NotContains(t, "Hello World", "Earth")
//	s.NotContains(t, ["Hello", "World"], "Earth")
//	s.NotContains(t, {"Hello": "World"}, "Earth")
func (s *Suite) NotContains(el, contains interface{}, msgAndArgs ...interface{}) {
	if ok := assert.NotContains(s.T(), el, contains); !ok {
		s.fail(KV{"list": s.f(contains), "item": s.f(el)}, "contains", msgAndArgs...)
	}
}

// Subset asserts that the specified list(array, slice...) contains all
// elements given in the specified subset(array, slice...).
//
//	s.Subset(t, [1, 2, 3], [1, 2], "But [1, 2, 3] does contain [1, 2]")
func (s *Suite) Subset(list, subset interface{}, msgAndArgs ...interface{}) {
	if ok := assert.Subset(s.T(), list, subset); !ok {
		s.fail(KV{"list": s.f(list), "subset": s.f(subset)}, "not subset", msgAndArgs...)
	}
}

// NotSubset asserts that the specified list(array, slice...) contains not all
// elements given in the specified subset(array, slice...).
//
//	s.NotSubset(t, [1, 3, 4], [1, 2], "But [1, 3, 4] does not contain [1, 2]")
func (s *Suite) NotSubset(list, subset interface{}, msgAndArgs ...interface{}) {
	if ok := assert.NotSubset(s.T(), list, subset); !ok {
		s.fail(KV{"list": s.f(list), "subset": s.f(subset)}, "subset", msgAndArgs...)
		s.T().FailNow()
	}
}

// ElementsMatch asserts that the specified listA(array, slice...) is equal to specified
// listB(array, slice...) ignoring the order of the elements. If there are duplicate elements,
// the number of appearances of each of them in both lists should match.
//
// s.ElementsMatch(t, [1, 3, 2, 3], [1, 3, 3, 2])
func (s *Suite) ElementsMatch(listA, listB interface{}, msgAndArgs ...interface{}) {
	if ok := assert.ElementsMatch(s.T(), listA, listB); !ok {
		s.fail(KV{"list": s.f(listA), "subset": s.f(listB)}, "lists don't match", msgAndArgs...)
	}
}

// Condition uses a Comparison to assert a complex condition.
func (s *Suite) Condition(comp assert.Comparison, msgAndArgs ...interface{}) {
	if ok := assert.Condition(s.T(), comp); !ok {
		s.fail(nil, "condition fail", msgAndArgs...)
	}
}

// Panics asserts that the code inside the specified PanicTestFunc panics.
//
//	s.Panics(t, func(){ GoCrazy() })
func (s *Suite) Panics(f assert.PanicTestFunc, msgAndArgs ...interface{}) {
	if ok := assert.Panics(s.T(), f); !ok {
		s.fail(nil, "should panics", msgAndArgs...)
	}
}

// PanicsWithValue asserts that the code inside the specified PanicTestFunc panics, and that
// the recovered panic value equals the expected panic value.
//
//	s.PanicsWithValue(t, "crazy error", func(){ GoCrazy() })
func (s *Suite) PanicsWithValue(expected interface{}, f assert.PanicTestFunc, msgAndArgs ...interface{}) {
	if ok := assert.PanicsWithValue(s.T(), expected, f); !ok {
		s.fail(KV{"panic value": s.f(expected)}, "should panics", msgAndArgs...)
	}
}

// PanicsWithError asserts that the code inside the specified PanicTestFunc
// panics, and that the recovered panic value is an error that satisfies the
// EqualError comparison.
//
//	s.PanicsWithError(t, "crazy error", func(){ GoCrazy() })
func (s *Suite) PanicsWithError(errString string, f assert.PanicTestFunc, msgAndArgs ...interface{}) {
	if ok := assert.PanicsWithError(s.T(), errString, f); !ok {
		s.fail(KV{"panic error": s.f(errString)}, "should panics", msgAndArgs...)
	}
}

// NotPanics asserts that the code inside the specified PanicTestFunc does NOT panic.
//
//	s.NotPanics(t, func(){ RemainCalm() })
func (s *Suite) NotPanics(f assert.PanicTestFunc, msgAndArgs ...interface{}) {
	if ok := assert.NotPanics(s.T(), f); !ok {
		s.fail(nil, "should not panics", msgAndArgs...)
	}
}

// WithinDuration asserts that the two times are within duration delta of each other.
//
//	s.WithinDuration(t, time.Now(), time.Now(), 10*time.Second)
func (s *Suite) WithinDuration(expected, actual time.Time, delta time.Duration, msgAndArgs ...interface{}) {
	if ok := assert.WithinDuration(s.T(), expected, actual, delta); !ok {
		s.fail(KV{"expected": expected, "actual": actual, "delta": delta}, "interval between expected and actual is longer than delta", msgAndArgs...)
	}
}

func toFloat(x interface{}) (float64, bool) {
	var xf float64
	xok := true

	switch xn := x.(type) {
	case uint:
		xf = float64(xn)
	case uint8:
		xf = float64(xn)
	case uint16:
		xf = float64(xn)
	case uint32:
		xf = float64(xn)
	case uint64:
		xf = float64(xn)
	case int:
		xf = float64(xn)
	case int8:
		xf = float64(xn)
	case int16:
		xf = float64(xn)
	case int32:
		xf = float64(xn)
	case int64:
		xf = float64(xn)
	case float32:
		xf = float64(xn)
	case float64:
		xf = xn
	case time.Duration:
		xf = float64(xn)
	default:
		xok = false
	}

	return xf, xok
}

// InDelta asserts that the two numerals are within delta of each other.
//
//	s.InDelta(t, math.Pi, 22/7.0, 0.01)
func (s *Suite) InDelta(expected, actual interface{}, delta float64, msgAndArgs ...interface{}) {
	af, aok := toFloat(expected)
	bf, bok := toFloat(actual)

	if !aok || !bok {
		s.Fail("Parameters must be numerical", msgAndArgs...)
	}

	if math.IsNaN(af) {
		s.Fail("Expected must not be NaN", msgAndArgs...)
	}

	if math.IsNaN(bf) {
		s.Fail(fmt.Sprintf("Expected %v with delta %v, but was NaN", expected, delta), msgAndArgs...)
	}

	dt := af - bf
	if dt < -delta || dt > delta {
		s.fail(KV{"expected": expected, "actual": actual, "delta": delta}, fmt.Sprintf("max difference between %v and %v allowed is %v, but difference was %v", expected, actual, delta, dt), msgAndArgs...)
	}
}

// InDeltaSlice is the same as InDelta, except it compares two slices.
func (s *Suite) InDeltaSlice(expected, actual interface{}, delta float64, msgAndArgs ...interface{}) {
	if expected == nil || actual == nil ||
		reflect.TypeOf(actual).Kind() != reflect.Slice ||
		reflect.TypeOf(expected).Kind() != reflect.Slice {
		s.Fail("Parameters must be slice", msgAndArgs...)
	}

	actualSlice := reflect.ValueOf(actual)
	expectedSlice := reflect.ValueOf(expected)

	for i := 0; i < actualSlice.Len(); i++ {
		s.InDelta(actualSlice.Index(i).Interface(), expectedSlice.Index(i).Interface(), delta, msgAndArgs...)
	}
}

// InDeltaMapValues is the same as InDelta, but it compares all values between two maps. Both maps must have exactly the same keys.
func (s *Suite) InDeltaMapValues(expected, actual interface{}, delta float64, msgAndArgs ...interface{}) {

	if expected == nil || actual == nil ||
		reflect.TypeOf(actual).Kind() != reflect.Map ||
		reflect.TypeOf(expected).Kind() != reflect.Map {
		s.Fail("Arguments must be maps", msgAndArgs...)
	}

	expectedMap := reflect.ValueOf(expected)
	actualMap := reflect.ValueOf(actual)

	if expectedMap.Len() != actualMap.Len() {
		s.Fail("Arguments must have the same number of keys", msgAndArgs...)
	}

	for _, k := range expectedMap.MapKeys() {
		ev := expectedMap.MapIndex(k)
		av := actualMap.MapIndex(k)

		if !ev.IsValid() {
			s.Fail(fmt.Sprintf("missing key %q in expected map", k), msgAndArgs...)
		}

		if !av.IsValid() {
			s.Fail(fmt.Sprintf("missing key %q in actual map", k), msgAndArgs...)
		}

		s.InDelta(ev.Interface(), av.Interface(), delta, msgAndArgs...)
	}
}

func calcRelativeError(expected, actual interface{}) (float64, error) {
	af, aok := toFloat(expected)
	if !aok {
		return 0, fmt.Errorf("expected value %q cannot be converted to float", expected)
	}
	if math.IsNaN(af) {
		return 0, errors.New("expected value must not be NaN")
	}
	if af == 0 {
		return 0, fmt.Errorf("expected value must have a value other than zero to calculate the relative error")
	}
	bf, bok := toFloat(actual)
	if !bok {
		return 0, fmt.Errorf("actual value %q cannot be converted to float", actual)
	}
	if math.IsNaN(bf) {
		return 0, errors.New("actual value must not be NaN")
	}

	return math.Abs(af-bf) / math.Abs(af), nil
}

// InEpsilon asserts that expected and actual have a relative error less than epsilon
func (s *Suite) InEpsilon(expected, actual interface{}, epsilon float64, msgAndArgs ...interface{}) {

	if math.IsNaN(epsilon) {
		s.Fail("epsilon must not be NaN")
	}
	actualEpsilon, err := calcRelativeError(expected, actual)
	if err != nil {
		s.Fail(err.Error(), msgAndArgs...)
	}
	if actualEpsilon > epsilon {
		s.fail(KV{"epsilon": epsilon, "actualEpsilon": actualEpsilon}, "relative error is too high", msgAndArgs...)

	}
}

// InEpsilonSlice is the same as InEpsilon, except it compares each value from two slices.
func (s *Suite) InEpsilonSlice(expected, actual interface{}, epsilon float64, msgAndArgs ...interface{}) {

	if expected == nil || actual == nil ||
		reflect.TypeOf(actual).Kind() != reflect.Slice ||
		reflect.TypeOf(expected).Kind() != reflect.Slice {
		s.Fail("Parameters must be slice", msgAndArgs...)
	}

	actualSlice := reflect.ValueOf(actual)
	expectedSlice := reflect.ValueOf(expected)

	for i := 0; i < actualSlice.Len(); i++ {
		s.InEpsilon(actualSlice.Index(i).Interface(), expectedSlice.Index(i).Interface(), epsilon)
	}
}

// Error asserts that a function returned an error (i.e. not `nil`).
//
//	  actualObj, err := SomeFunction()
//	  if s.Error(t, err) {
//		   s.Equal(t, expectedError, err)
//	  }
func (s *Suite) Error(err error, msgAndArgs ...interface{}) {
	if ok := assert.Error(s.T(), err); !ok {
		s.fail(nil, "not error", msgAndArgs...)
	}
}

// NoError asserts that a function returned no error (i.e. `nil`).
//
//	  actualObj, err := SomeFunction()
//	  if s.NoError(t, err) {
//		   s.Equal(t, expectedObj, actualObj)
//	  }
func (s *Suite) NoError(err error, msgAndArgs ...interface{}) {
	if ok := assert.NoError(s.T(), err); !ok {
		s.fail(KV{"error": err}, "error", msgAndArgs...)
	}
}

// EqualError asserts that a function returned an error (i.e. not `nil`)
// and that it is equal to the provided error.
//
//	actualObj, err := SomeFunction()
//	s.EqualError(t, err,  expectedErrorString)
func (s *Suite) EqualError(theError error, errString string, msgAndArgs ...interface{}) {
	if ok := assert.EqualError(s.T(), theError, errString); !ok {
		s.fail(KV{"error": theError, "errString": errString}, "not equal errors", msgAndArgs...)
	}
}

// Regexp asserts that a specified regexp matches a string.
//
//	s.Regexp(t, regexp.MustCompile("start"), "it's starting")
//	s.Regexp(t, "start...$", "it's not starting")
func (s *Suite) Regexp(rx interface{}, str interface{}, msgAndArgs ...interface{}) {
	if ok := assert.Regexp(s.T(), rx, str); !ok {
		s.fail(KV{"regex": rx, "string": str}, "str not match regexp", msgAndArgs...)
	}
}

// NotRegexp asserts that a specified regexp does not match a string.
//
//	s.NotRegexp(t, regexp.MustCompile("starts"), "it's starting")
//	s.NotRegexp(t, "^start", "it's not starting")
func (s *Suite) NotRegexp(rx interface{}, str interface{}, msgAndArgs ...interface{}) {
	if ok := assert.NotRegexp(s.T(), rx, str); !ok {
		s.fail(KV{"regex": rx, "string": str}, "str match regexp", msgAndArgs...)
	}
}

// Zero asserts that i is the zero value for its type.
func (s *Suite) Zero(v interface{}, msgAndArgs ...interface{}) {
	if ok := assert.Zero(s.T(), v); !ok {
		s.fail(KV{"value": v}, "not a zero value", msgAndArgs...)
	}
}

// NotZero asserts that i is not the zero value for its type.
func (s *Suite) NotZero(v interface{}, msgAndArgs ...interface{}) {
	if ok := assert.NotZero(s.T(), v); !ok {
		s.fail(KV{"value": v}, "zero value", msgAndArgs...)
	}
}

// FileExists checks whether a file exists in the given path. It also fails if
// the path points to a directory or there is an error when trying to check the file.
func (s *Suite) FileExists(path string, msgAndArgs ...interface{}) {
	if ok := assert.FileExists(s.T(), path); !ok {
		s.fail(KV{"path": path}, "file not exists", msgAndArgs...)
	}
}

// NoFileExists checks whether a file does not exist in a given path. It fails
// if the path points to an existing _file_ only.
func (s *Suite) NoFileExists(path string, msgAndArgs ...interface{}) {
	if ok := assert.NoFileExists(s.T(), path); !ok {
		s.fail(KV{"path": path}, "file exists", msgAndArgs...)
	}
}

// DirExists checks whether a directory exists in the given path. It also fails
// if the path is a file rather a directory or there is an error checking whether it exists.
func (s *Suite) DirExists(dir string, msgAndArgs ...interface{}) {
	if ok := assert.DirExists(s.T(), dir); !ok {
		s.fail(KV{"dir": dir}, "dir not exists", msgAndArgs...)
	}
}

// NoDirExists checks whether a directory does not exist in the given path.
// It fails if the path points to an existing _directory_ only.
func (s *Suite) NoDirExists(dir string, msgAndArgs ...interface{}) {
	if ok := assert.NoDirExists(s.T(), dir); !ok {
		s.fail(KV{"dir": dir}, "dir exists", msgAndArgs...)
	}
}

// JSONEq asserts that two JSON strings are equivalent.
//
//	s.JSONEq(t, `{"hello": "world", "foo": "bar"}`, `{"foo": "bar", "hello": "world"}`)
func (s *Suite) JSONEq(expected string, actual string, msgAndArgs ...interface{}) {
	if ok := assert.JSONEq(s.T(), expected, actual); !ok {
		s.fail(KV{"expected": expected, "actual": actual}, "not equal json", msgAndArgs...)
	}
}

// YAMLEq asserts that two YAML strings are equivalent.
func (s *Suite) YAMLEq(expected string, actual string, msgAndArgs ...interface{}) {
	if ok := assert.YAMLEq(s.T(), expected, actual); !ok {
		s.fail(KV{"expected": expected, "actual": actual}, "not equal yaml", msgAndArgs...)
	}
}

// Eventually asserts that given condition will be met in waitFor time,
// periodically checking target function each tick.
//
//	s.Eventually(t, func() bool { return true; }, time.Second, 10*time.Millisecond)
func (s *Suite) Eventually(condition func() bool, waitFor time.Duration, tick time.Duration, msgAndArgs ...interface{}) {
	if ok := assert.Eventually(s.T(), condition, waitFor, tick, msgAndArgs); !ok {
		s.fail(nil, "condition never satisfied", msgAndArgs...)
	}
}

// Never asserts that the given condition doesn't satisfy in waitFor time,
// periodically checking the target function each tick.
//
//	s.Never(t, func() bool { return false; }, time.Second, 10*time.Millisecond)
func (s *Suite) Never(condition func() bool, waitFor time.Duration, tick time.Duration, msgAndArgs ...interface{}) {
	if ok := assert.Never(s.T(), condition, waitFor, tick, msgAndArgs); !ok {
		s.fail(nil, "condition satisfied", msgAndArgs...)
	}
}

// ErrorIs asserts that at least one of the errors in err's chain matches target.
// This is a wrapper for errors.IsAppErr.
func (s *Suite) ErrorIs(err, target error, msgAndArgs ...interface{}) {
	if ok := assert.ErrorIs(s.T(), err, target, msgAndArgs); !ok {

		var expectedText string
		if target != nil {
			expectedText = target.Error()
		}

		chain := buildErrorChainString(err)

		s.fail(KV{"expected": expectedText, "chain": chain}, "target error should be in err chain", msgAndArgs...)
	}
}

// NotErrorIs asserts that at none of the errors in err's chain matches target.
// This is a wrapper for errors.IsAppErr.
func (s *Suite) NotErrorIs(err, target error, msgAndArgs ...interface{}) {
	if ok := assert.NotErrorIs(s.T(), err, target, msgAndArgs); !ok {

		var expectedText string
		if target != nil {
			expectedText = target.Error()
		}

		chain := buildErrorChainString(err)

		s.fail(KV{"expected": expectedText, "chain": chain}, "target error should not be in err chain", msgAndArgs...)
	}
}

// ErrorAs asserts that at least one of the errors in err's chain matches target, and if so, sets target to that error value.
// This is a wrapper for errors.As.
func (s *Suite) ErrorAs(err, target error, msgAndArgs ...interface{}) {
	if ok := assert.ErrorAs(s.T(), err, target, msgAndArgs); !ok {

		var expectedText string
		if target != nil {
			expectedText = target.Error()
		}

		chain := buildErrorChainString(err)

		s.fail(KV{"expected": expectedText, "chain": chain}, "target error should not be in err chain", msgAndArgs...)
	}
}

func buildErrorChainString(err error) string {
	if err == nil {
		return ""
	}

	e := errors.Unwrap(err)
	chain := fmt.Sprintf("%q", err.Error())
	for e != nil {
		chain += fmt.Sprintf("\n\t%q", e.Error())
		e = errors.Unwrap(e)
	}
	return chain
}

// AssertNotCalled asserts that the method was not called.
// It can produce a false result when an argument is a pointer type and the underlying value changed after calling the mocked method.
func (s *Suite) AssertNotCalled(mock *mock.Mock, methodName string, arguments ...interface{}) {
	if !mock.AssertNotCalled(s.T(), methodName, arguments...) {
		s.fail(KV{"method": methodName}, "assert not called", arguments...)
	}
}

// AssertCalled asserts that the method was called.
// It can produce a false result when an argument is a pointer type and the underlying value changed after calling the mocked method.
func (s *Suite) AssertCalled(mock *mock.Mock, methodName string, arguments ...interface{}) {
	if !mock.AssertCalled(s.T(), methodName, arguments...) {
		s.fail(KV{"method": methodName}, "assert called", arguments...)
	}
}

// AssertNumberOfCalls asserts that the method was called expectedCalls times.
func (s *Suite) AssertNumberOfCalls(mock *mock.Mock, methodName string, expectedCalls int) {
	if !mock.AssertNumberOfCalls(s.T(), methodName, expectedCalls) {
		s.fail(KV{"method": methodName, "expected": expectedCalls}, "assert number of calls")
	}
}

func (s *Suite) RandPhone() string {
	return strconv.Itoa(int(rand.Int31n(1000000000)))
}
