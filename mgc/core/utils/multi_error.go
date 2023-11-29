package utils

import (
	"fmt"
	"strings"
)

type MultiError []error

var _ error = (*MultiError)(nil)

func (e MultiError) Unwrap() []error {
	return []error(e)
}

func (e MultiError) Error() string {
	switch n := len(e); n {
	case 0:
		panic("programming error: must never return empty errors")

	case 1:
		return e[0].Error()

	default:
		s := fmt.Sprintf("%d errors:", n)
		for i, sub := range []error(e) {
			m := strings.Join(strings.Split(sub.Error(), "\n"), "\n\t\t")
			s += fmt.Sprintf("\n\terror #%d: %s", i, m)
		}
		return s
	}
}
