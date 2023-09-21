package cmd

import (
	"reflect"
	"testing"
)

type testCase[T any] struct {
	separator T
	slice     []T
	expected  [][]T
}

var stringTests = []testCase[string]{
	{
		separator: "2",
		slice:     []string{"some", "string", "2", "test", "case"},
		expected:  [][]string{{"some", "string"}, {"test", "case"}},
	},
	{
		separator: "2",
		slice:     []string{"some", "string", "2", "test", "case", "2"},
		expected:  [][]string{{"some", "string"}, {"test", "case"}},
	},
	{
		separator: "2",
		slice:     []string{"2", "some", "string", "2", "test", "case", "2", "2", "2"},
		expected:  [][]string{{"some", "string"}, {"test", "case"}},
	},
	{
		separator: "sep",
		slice:     []string{"sep", "string", "sep", "sep", "case", "2"},
		expected:  [][]string{{"string"}, {"case", "2"}},
	},
	{
		separator: "sep",
		slice:     []string{"some", "string", "something", "123", "case", "2"},
		expected:  [][]string{{"some", "string", "something", "123", "case", "2"}},
	},
	{
		separator: "sep",
		slice:     []string{"sep", "sep", "sep", "sep"},
		expected:  [][]string{},
	},
	{
		separator: "sep",
		slice:     []string{},
		expected:  [][]string{},
	},
	{
		separator: "sep",
		slice:     nil,
		expected:  nil,
	},
}

func TestSplitSlice(t *testing.T) {
	for _, test := range stringTests {
		result := splitSlice(test.slice, test.separator)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("slice split failed\ninput: %v separator: %v\nexpected: %v\ngot: %v", test.slice, test.separator, test.expected, result)
		}
	}
}
