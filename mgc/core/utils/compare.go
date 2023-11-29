package utils

import (
	"fmt"
	"reflect"

	"slices"
)

// Checks if the value is exactly the same, it's basically a "=="
// but handles map/slice and other non-comparable types as pointers,
// that is if their underlying pointer is the same.
// It DOES NOT check for map/slice similarities, only their addresses.
func IsSameValueOrPointer(a, b any) bool {
	if a == nil {
		return b == nil
	} else if b == nil {
		return false
	}

	vA := reflect.ValueOf(a)
	vB := reflect.ValueOf(b)
	if vA.Type() != vB.Type() {
		return false
	}

	vA = valueDereference(vA)
	vB = valueDereference(vB)

	if vA.Comparable() {
		return vA.Interface() == vB.Interface()
	} else {
		switch vA.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan, reflect.UnsafePointer, reflect.Func:
			return vA.UnsafePointer() == vB.UnsafePointer()
		default:
			return reflect.DeepEqual(a, b)
		}
	}
}

func IsComparableEqual[V comparable](a, b V) bool {
	return a == b
}

func IsComparablePointerEqual[V comparable](a, b *V) bool {
	return IsPointerEqualFunc[V](a, b, func(v1, v2 *V) bool {
		return IsComparableEqual[V](*a, *b)
	})
}

// equals is only called if both pointers are non-nil
func IsPointerEqualFunc[V any](a, b *V, equals func(*V, *V) bool) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return equals(a, b)
}

func valueDereference(value reflect.Value) reflect.Value {
	for value.Kind() == reflect.Pointer && !value.IsNil() {
		value = value.Elem()
	}
	return value
}

func ReflectValueDeepEqualCompare(vA, vB reflect.Value) (err error) {
	a := vA.Interface()
	b := vB.Interface()
	if reflect.DeepEqual(a, b) {
		return nil
	}

	return &CompareError{A: a, B: b}
}

func UnorderedSliceDeepEqual[T any, S ~[]T](a, b S) bool {
	return UnorderedSliceCompareDeepEqual(a, b) == nil
}

func UnorderedSliceCompareDeepEqual[T any, S ~[]T](a, b S) (err error) {
	return ReflectValueUnorderedSliceCompareDeepEqual(
		reflect.ValueOf(a),
		reflect.ValueOf(b),
	)
}

type ReflectValueCompareFn func(a, b reflect.Value) error

func ReflectValueUnorderedSliceCompareDeepEqual(vA, vB reflect.Value) (err error) {
	return ReflectValueUnorderedSliceCompare(vA, vB, ReflectValueDeepEqualCompare)
}

var _ ReflectValueCompareFn = ReflectValueUnorderedSliceCompareDeepEqual

const (
	CompareTypeErrorKey   = "$type"
	CompareLengthErrorKey = "$length"
	CompareAErrorKey      = "$A"
	CompareBErrorKey      = "$B"
)

// Compares the slices ignoring their order (quadratic: O(len(a) * O(len(b)))
func ReflectValueUnorderedSliceCompare(vA, vB reflect.Value, compare ReflectValueCompareFn) (err error) {
	if vA.Type() != vB.Type() {
		return &ChainedError{Name: CompareTypeErrorKey, Err: &CompareError{A: vA.Interface(), B: vB.Interface()}}
	}
	if vA.Kind() != reflect.Slice && vA.Kind() != reflect.Array {
		return &ChainedError{Name: CompareTypeErrorKey, Err: fmt.Errorf("expected slice, got %s", vA)}
	}

	aLen := vA.Len()
	bLen := vB.Len()
	if aLen != bLen {
		return &ChainedError{Name: CompareLengthErrorKey, Err: &CompareError{A: aLen, B: bLen}}
	}

	size := bLen
	if size == 0 {
		return nil
	}

	pending := make([]reflect.Value, size)
	for i := 0; i < size; i++ {
		pending[i] = vB.Index(i)
	}

	var aMissing []any
	for i := 0; i < size; i++ {
		aElement := vA.Index(i)
		foundIndex := slices.IndexFunc(pending, func(bElement reflect.Value) bool {
			return compare(aElement, bElement) == nil
		})
		if foundIndex < 0 {
			aMissing = append(aMissing, aElement.Interface())
		} else {
			pending = slices.Delete(pending, foundIndex, foundIndex+1)
		}
	}

	if len(pending) == 0 && len(aMissing) == 0 {
		return nil
	}

	var bMissing []any
	for _, bElement := range pending {
		bMissing = append(bMissing, bElement.Interface())
	}

	msg := "missing elements: "
	if len(aMissing) > 0 {
		msg += fmt.Sprintf("A=%v", aMissing)
	}
	if len(aMissing) > 0 && len(bMissing) > 0 {
		msg += ", "
	}
	if len(bMissing) > 0 {
		msg += fmt.Sprintf("B=%v", bMissing)
	}

	return &CompareError{A: vA.Interface(), B: vB.Interface(), Message: msg}
}

type StructFieldsComparator interface {
	Type() reflect.Type
	// field index at reflect.Type
	Compare(field int, vA, vB reflect.Value) (err error)
}

type mapStructFieldsComparator struct {
	t       reflect.Type
	compare []ReflectValueCompareFn
}

func (c *mapStructFieldsComparator) Type() reflect.Type {
	return c.t
}

func (c *mapStructFieldsComparator) Compare(field int, vA, vB reflect.Value) (err error) {
	cmp := c.compare[field]
	if cmp != nil {
		return cmp(vA, vB)
	}
	return nil
}

var _ StructFieldsComparator = (*mapStructFieldsComparator)(nil)

func NewMapStructFieldsComparator[T any](fields map[string]ReflectValueCompareFn) StructFieldsComparator {
	if len(fields) == 0 {
		panic("NewMapStructFieldsComparator: programming error: needs non-nil fields")
	}

	t := reflect.TypeOf(*new(T))
	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("NewMapStructFieldsComparator: programming error: expected struct, got %s", t))
	}

	fieldCompare := make([]ReflectValueCompareFn, t.NumField())
	for name := range fields {
		if f, ok := t.FieldByName(name); !ok {
			panic("NewMapStructFieldsComparator: programming error: unknown field name: " + name)
		} else if !f.IsExported() {
			panic("NewMapStructFieldsComparator: programming error: private field name: " + name)
		} else {
			fieldCompare[f.Index[0]] = fields[f.Name]
		}
	}

	return &mapStructFieldsComparator{t, fieldCompare}
}

func StructFieldsEqual[T any](a, b T, cmp StructFieldsComparator) bool {
	return StructFieldsCompare[T](a, b, cmp) == nil
}

func StructFieldsCompare[T any](a, b T, cmp StructFieldsComparator) (err error) {
	return ReflectValueStructFieldsCompare(
		reflect.ValueOf(a),
		reflect.ValueOf(b),
		cmp,
	)
}

func ReflectValueStructFieldsCompare(vA, vB reflect.Value, cmp StructFieldsComparator) (err error) {
	t := cmp.Type()
	vA = valueDereference(vA)
	vB = valueDereference(vB)

	if vA.Type() != t {
		return &ChainedError{Name: CompareTypeErrorKey, Err: &ChainedError{Name: CompareAErrorKey, Err: &CompareError{A: vA.Type(), B: t}}}
	}

	if vB.Type() != t {
		return &ChainedError{Name: CompareTypeErrorKey, Err: &ChainedError{Name: CompareBErrorKey, Err: &CompareError{A: vB.Type(), B: t}}}
	}

	nFields := t.NumField()
	for i := 0; i < nFields; i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		aValue := vA.Field(i)
		bValue := vB.Field(i)
		err = cmp.Compare(i, aValue, bValue)
		if err != nil {
			return &ChainedError{Name: f.Name, Err: err}
		}
	}

	return nil
}
