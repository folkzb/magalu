package utils

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"golang.org/x/exp/slices"
)

func checkCompareError(t *testing.T, prefix string, err error, wantError error) {
	if !errors.Is(err, wantError) {
		t.Errorf("%s: expected\n%#v\n%s\ngot:\n%#v\n%s", prefix, wantError, wantError.Error(), err, err.Error())
	}
}

func checkUnorderedSliceCompareDeepEqual[T any](t *testing.T, a, b []T, missingValue T) {
	missing := slices.Clone(a)
	missing[len(a)/2] = missingValue
	empty := []T{}

	larger := make([]T, len(a)+1)

	type compareTestCase[T any] struct {
		name string
		a, b []T
		err  error
	}

	tests := []compareTestCase[T]{
		{name: "equal/same", a: a, b: a},
		{name: "equal/different order", a: a, b: b},
		{name: "equal/both empty", a: empty, b: empty},
		{name: "equal/both nil", a: nil, b: nil},
		{name: "error/different size", a: a, b: larger, err: &ChainedError{Name: CompareLengthErrorKey, Err: &CompareError{A: len(a), B: len(larger)}}},
		{name: "error/missing a", a: missing, b: b, err: &CompareError{A: missing, B: b}},
		{name: "error/missing b", a: a, b: missing, err: &CompareError{A: a, B: missing}},
		{name: "error/empty a", a: empty, b: b, err: &ChainedError{Name: CompareLengthErrorKey, Err: &CompareError{A: len(empty), B: len(b)}}},
		{name: "error/empty b", a: a, b: empty, err: &ChainedError{Name: CompareLengthErrorKey, Err: &CompareError{A: len(a), B: len(empty)}}},
		{name: "error/nil a", a: nil, b: b, err: &ChainedError{Name: CompareLengthErrorKey, Err: &CompareError{A: 0, B: len(b)}}},
		{name: "error/nil b", a: a, b: nil, err: &ChainedError{Name: CompareLengthErrorKey, Err: &CompareError{A: len(a), B: 0}}},
	}

	prefix := fmt.Sprintf("%T", *new(T))

	for _, tc := range tests {
		name := fmt.Sprintf("%s/%s", prefix, tc.name)
		t.Run(tc.name, func(t *testing.T) {
			err := UnorderedSliceCompareDeepEqual(tc.a, tc.b)
			if tc.err != nil {
				checkCompareError(t, name, err, tc.err)
			} else if tc.err == nil && err != nil {
				t.Errorf("%s: unexpected error: %s", name, err.Error())
			}
		})
	}
}

func Test_UnorderedSliceCompareDeepEqual_int(t *testing.T) {
	checkUnorderedSliceCompareDeepEqual[int](
		t,
		[]int{1, 2, 3},
		[]int{3, 2, 1},
		4,
	)
}

func Test_UnorderedSliceCompareDeepEqual_struct(t *testing.T) {
	type testSt struct {
		s string
		i int
	}
	checkUnorderedSliceCompareDeepEqual[testSt](
		t,
		[]testSt{{"a", 1}, {"b", 2}, {"c", 3}},
		[]testSt{{"c", 3}, {"b", 2}, {"a", 1}},
		testSt{"d", 4},
	)
}

func Test_UnorderedSliceCompareDeepEqual_map(t *testing.T) {
	checkUnorderedSliceCompareDeepEqual[map[string]int](
		t,
		[]map[string]int{{"a": 1}, {"b": 2}, {"c": 3}},
		[]map[string]int{{"c": 3}, {"b": 2}, {"a": 1}},
		map[string]int{"d": 4},
	)
}

func Test_ReflectValueUnorderedSliceCompareDeepEqual_DifferentType(t *testing.T) {
	a := []string{"a"}
	b := []int{2}
	err := ReflectValueUnorderedSliceCompareDeepEqual(
		reflect.ValueOf(a),
		reflect.ValueOf(b),
	)
	prefix := "ReflectValueUnorderedSliceCompareDeepEqual"
	checkCompareError(t, prefix, err, &ChainedError{Name: CompareTypeErrorKey, Err: &CompareError{A: a, B: b}})
}

func Test_ReflectValueUnorderedSliceCompareDeepEqual_InvalidType(t *testing.T) {
	err := ReflectValueUnorderedSliceCompareDeepEqual(
		reflect.ValueOf(1),
		reflect.ValueOf(2),
	)
	prefix := "ReflectValueUnorderedSliceCompareDeepEqual"
	var chainedError *ChainedError
	if !errors.As(err, &chainedError) {
		t.Errorf("%s: expected ChainedError, got %#v", prefix, err)
	} else if chainedError.Name != CompareTypeErrorKey {
		t.Errorf("%s: expected ChainedError.Name == %q, got %#v", prefix, CompareTypeErrorKey, chainedError)
	}
}
