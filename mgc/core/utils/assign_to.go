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

	tVal := reflect.ValueOf(t).Elem()
	// Empty name means `any`
	if tVal.Type().Name() != "" {
		return fmt.Errorf("request response of type %T is not convertible to %T", *t, u)
	}

	tVal.Set(reflect.ValueOf(u))
	return nil
}
