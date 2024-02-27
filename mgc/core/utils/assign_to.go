package utils

import (
	"fmt"
	"reflect"
)

func AssignToT[T any, U any](t *T, u U) error {
	if uAsT, ok := any(u).(T); ok {
		*t = uAsT
		return nil
	}

	if t == nil {
		return fmt.Errorf("can't assign value %#v to nil pointer", u)
	}

	tVal := reflect.ValueOf(t).Elem()
	// Empty 'String()' means `any`
	if tVal.Type().String() != "" {
		return fmt.Errorf("request response of type %T is not convertible to %T", u, *t)
	}

	tVal.Set(reflect.ValueOf(u))
	return nil
}
