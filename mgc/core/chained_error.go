package core

import (
	"errors"
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

// Chained error that builds a path in the error message
//
// It walks Err chain looking for other ChainedError,
// building a JSON Pointer-compatible path ("/" and "~" are escaped with "~0/1")
// and outputs the final error message
//
// It's particularly useful to validate specifications and nested trees.
type ChainedError struct {
	Name string
	Err  error
}

func (e *ChainedError) Error() string {
	path := ""
	n := e
	var leaf error
	for {
		name := n.Name
		if name == "" {
			name = "<unnamed>"
		}
		path += "/" + jsonpointer.Escape(name)
		leaf = n.Err
		if ok := errors.As(leaf, &n); !ok {
			break
		}
	}
	return fmt.Sprintf("invalid %q: %s", path, leaf.Error())
}

func (e *ChainedError) Unwrap() error {
	return e.Err
}

var _ error = (*ChainedError)(nil)
