package utils

import (
	"fmt"
)

type CompareError struct {
	A, B    any
	Op      string
	Message string
}

func (e *CompareError) Error() string {
	if e.Message != "" {
		return e.Message
	}

	op := e.Op
	if op == "" {
		op = "!="
	}
	return fmt.Sprintf("%#v %s %#v", e.A, op, e.B)
}

func (e *CompareError) Is(target error) bool {
	if other, ok := target.(*CompareError); ok {
		if e.Message != "" && other.Message != "" && e.Message != other.Message {
			return false
		}
		return e.Op == other.Op &&
			IsSameValueOrPointer(e.A, other.A) &&
			IsSameValueOrPointer(e.B, other.B)
	}
	return false
}

var _ error = (*CompareError)(nil)
