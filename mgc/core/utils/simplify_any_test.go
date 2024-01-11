package utils

import (
	"reflect"
	"testing"

	"slices"
)

type testCase struct {
	Name          string
	Input, Output interface{}
}

type simpleStruct struct {
	StrField  string
	IntField  int `json:"int_field_with_json_name"`
	BoolField bool
}

type complexStruct struct {
	StructField    simpleStruct
	StructPtrField *simpleStruct
	BoolField      bool
}

type CustomStr string
type CustomInt int
type CustomUInt uint
type CustomFloat float32

var strPtr = new(string)
var boolPtr = new(bool)
var int8Ptr = new(int8)
var int16Ptr = new(int16)
var int32Ptr = new(int32)
var int64Ptr = new(int64)
var uint8Ptr = new(uint8)
var uint16Ptr = new(uint16)
var uint32Ptr = new(uint32)
var uint64Ptr = new(uint64)
var float32Ptr = new(float32)
var float64Ptr = new(float64)

var customStrPtr = new(CustomStr)
var customIntPtr = new(CustomInt)
var customUIntPtr = new(CustomUInt)
var customFloatPtr = new(CustomFloat)

var dataResultArr = []testCase{
	{Name: "string ptr to string", Input: strPtr, Output: "hello"},
	{Name: "bool ptr to bool", Input: boolPtr, Output: true},
	{Name: "int8 ptr to int64", Input: int8Ptr, Output: int64(123)},
	{Name: "int16 ptr to int64", Input: int16Ptr, Output: int64(123)},
	{Name: "int32 ptr to int64", Input: int32Ptr, Output: int64(123)},
	{Name: "int64 ptr to int64", Input: int64Ptr, Output: int64(123)},
	{Name: "uint8 ptr to uint64", Input: uint8Ptr, Output: uint64(123)},
	{Name: "uint16 ptr to uint64", Input: uint16Ptr, Output: uint64(123)},
	{Name: "uint32 ptr to uint64", Input: uint32Ptr, Output: uint64(123)},
	{Name: "uint64 ptr to uint64", Input: uint64Ptr, Output: uint64(123)},
	{Name: "float32 ptr to float64", Input: float32Ptr, Output: float64(123.44999694824219)}, // account for imprecise conversion from 32 to 64
	{Name: "float64 ptr to float64", Input: float64Ptr, Output: float64(123.45)},
	{Name: "custom string ptr to string", Input: customStrPtr, Output: "hello"},
	{Name: "custom int ptr to string", Input: customIntPtr, Output: int64(123)},
	{Name: "custom uint ptr to string", Input: customUIntPtr, Output: uint64(123)},
	{Name: "custom float32 ptr to string", Input: customFloatPtr, Output: float64(123.44999694824219)}, // account for imprecise conversion from 32 to 64
	{Name: "untyped string to string", Input: "hello", Output: "hello"},
	{Name: "untyped int to int64", Input: 123, Output: int64(123)},
	{Name: "untyped float to float64", Input: 123.45, Output: 123.45},
	{Name: "string slice to any slice", Input: []string{"one", "two", "three"}, Output: []any{"one", "two", "three"}},
	{Name: "string ptr slice to any slice", Input: []*string{strPtr}, Output: []any{"hello"}},
	{Name: "int slice to any slice", Input: []int{1, 2, 3}, Output: []any{int64(1), int64(2), int64(3)}},
	{
		Name:   "simple struct to map[string]any",
		Input:  simpleStruct{StrField: "hello", IntField: 123, BoolField: true},
		Output: map[string]any{"StrField": "hello", "int_field_with_json_name": int64(123), "BoolField": true},
	},
	{
		Name:   "simple struct ptr to map[string]any",
		Input:  &simpleStruct{StrField: "hello", IntField: 123, BoolField: true},
		Output: map[string]any{"StrField": "hello", "int_field_with_json_name": int64(123), "BoolField": true},
	},
	{
		Name: "complex struct to map[string]any",
		Input: complexStruct{
			StructField:    simpleStruct{StrField: "hola", IntField: 321, BoolField: false},
			StructPtrField: &simpleStruct{StrField: "oi", IntField: 000, BoolField: true},
			BoolField:      false,
		},
		Output: map[string]any{
			"StructField":    map[string]any{"StrField": "hola", "int_field_with_json_name": int64(321), "BoolField": false},
			"StructPtrField": map[string]any{"StrField": "oi", "int_field_with_json_name": int64(000), "BoolField": true},
			"BoolField":      false,
		},
	},
	{
		Name: "complex struct ptr to map[string]any",
		Input: &complexStruct{
			StructField:    simpleStruct{StrField: "hola", IntField: 321, BoolField: false},
			StructPtrField: &simpleStruct{StrField: "oi", IntField: 000, BoolField: true},
			BoolField:      false,
		},
		Output: map[string]any{
			"StructField":    map[string]any{"StrField": "hola", "int_field_with_json_name": int64(321), "BoolField": false},
			"StructPtrField": map[string]any{"StrField": "oi", "int_field_with_json_name": int64(000), "BoolField": true},
			"BoolField":      false,
		},
	},
	{
		Name: "complex struct slice to []any",
		Input: []complexStruct{
			{
				StructField:    simpleStruct{StrField: "hola", IntField: 321, BoolField: false},
				StructPtrField: &simpleStruct{StrField: "oi", IntField: 000, BoolField: true},
				BoolField:      false,
			},
			{
				StructField:    simpleStruct{StrField: "ciao", IntField: 456, BoolField: true},
				StructPtrField: &simpleStruct{StrField: "oi", IntField: 999, BoolField: false},
				BoolField:      true,
			},
		},
		Output: []any{
			map[string]any{
				"StructField":    map[string]any{"StrField": "hola", "int_field_with_json_name": int64(321), "BoolField": false},
				"StructPtrField": map[string]any{"StrField": "oi", "int_field_with_json_name": int64(000), "BoolField": true},
				"BoolField":      false,
			},
			map[string]any{
				"StructField":    map[string]any{"StrField": "ciao", "int_field_with_json_name": int64(456), "BoolField": true},
				"StructPtrField": map[string]any{"StrField": "oi", "int_field_with_json_name": int64(999), "BoolField": false},
				"BoolField":      true,
			},
		},
	},
	{
		Name: "complex struct ptr slice to []any",
		Input: []*complexStruct{
			{
				StructField:    simpleStruct{StrField: "hola", IntField: 321, BoolField: false},
				StructPtrField: &simpleStruct{StrField: "oi", IntField: 000, BoolField: true},
				BoolField:      false,
			},
			{
				StructField:    simpleStruct{StrField: "ciao", IntField: 456, BoolField: true},
				StructPtrField: &simpleStruct{StrField: "oi", IntField: 999, BoolField: false},
				BoolField:      true,
			},
		},
		Output: []any{
			map[string]any{
				"StructField":    map[string]any{"StrField": "hola", "int_field_with_json_name": int64(321), "BoolField": false},
				"StructPtrField": map[string]any{"StrField": "oi", "int_field_with_json_name": int64(000), "BoolField": true},
				"BoolField":      false,
			},
			map[string]any{
				"StructField":    map[string]any{"StrField": "ciao", "int_field_with_json_name": int64(456), "BoolField": true},
				"StructPtrField": map[string]any{"StrField": "oi", "int_field_with_json_name": int64(999), "BoolField": false},
				"BoolField":      true,
			},
		},
	},
	{Input: func() {}, Output: nil},
}

var expectedErrors = []reflect.Kind{
	reflect.Invalid,
	reflect.Chan,
	reflect.Complex64,
	reflect.Complex128,
	reflect.Func,
	reflect.UnsafePointer,
	reflect.Interface,
}

func TestSimplifyAny(t *testing.T) {
	for _, testCase := range dataResultArr {
		simple, err := SimplifyAny(testCase.Input)
		if err != nil && !slices.Contains(expectedErrors, reflect.TypeOf(testCase.Input).Kind()) {
			t.Error(err)
		}
		if !reflect.DeepEqual(simple, testCase.Output) {
			t.Errorf("%s: Failed to convert value %#v, of type %T, to %#v. Got %#v instead", testCase.Name, testCase.Input, testCase.Input, testCase.Output, simple)
		}
	}
}

func init() {
	*strPtr = "hello"
	*boolPtr = true
	*int8Ptr = 123
	*int16Ptr = int16(123)
	*int32Ptr = int32(123)
	*int64Ptr = int64(123)
	*uint8Ptr = uint8(123)
	*uint16Ptr = uint16(123)
	*uint32Ptr = uint32(123)
	*uint64Ptr = uint64(123)
	*float32Ptr = float32(123.45)
	*float64Ptr = float64(123.45)

	*customStrPtr = "hello"
	*customIntPtr = CustomInt(123)
	*customUIntPtr = CustomUInt(123)
	*customFloatPtr = CustomFloat(123.45)
}
