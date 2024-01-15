package utils

import (
	"context"
	"fmt"

	"github.com/PaesslerAG/gval"
	"github.com/PaesslerAG/jsonpath"
)

var hasKeyFunc = gval.Function("hasKey", func(args ...any) (any, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("'hasKey' jsonpath function expects two arguments, a map[string]any and a string (key). Got %v instead", args...)
	}

	m, ok := args[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("'hasKey' jsonpath function expected map[string]any as first argument, got %T instead: %v", args[0], args[0])
	}

	k, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("'hasKey' jsonpath function expected string as second argument, got %T instead: %v", args[1], args[1])
	}

	_, hasKey := m[k]
	return hasKey, nil
})

var jsonPathBuilder = gval.Full(
	jsonpath.PlaceholderExtension(),
	hasKeyFunc,
)

func NewJsonPath(expression string) (jp gval.Evaluable, err error) {
	return jsonPathBuilder.NewEvaluable(expression)
}

func GetJsonPath(expression string, document any) (result any, err error) {
	jp, err := NewJsonPath(expression)
	if err != nil {
		return nil, err
	}
	return jp(context.Background(), document)
}

func CreateJsonPathChecker(expression string) (checker func(document any) (bool, error), err error) {
	jp, err := NewJsonPath(expression)
	if err != nil {
		return nil, err
	}
	return CreateJsonPathCheckerFromEvaluable(jp), nil
}

func CreateJsonPathCheckerFromEvaluable(jp gval.Evaluable) (checker func(document any) (bool, error)) {
	return func(value any) (ok bool, err error) {
		v, err := jp(context.Background(), value)
		if err != nil {
			return false, err
		}

		if v == nil {
			return false, nil
		} else if lst, ok := v.([]any); ok {
			return len(lst) > 0, nil
		} else if m, ok := v.(map[string]any); ok {
			return len(m) > 0, nil
		} else if b, ok := v.(bool); ok {
			return b, nil
		} else {
			return false, fmt.Errorf("unknown jsonpath result. Expected list, map or boolean. Got %#v", value)
		}
	}
}
