package utils

import (
	"context"
	"fmt"

	"github.com/PaesslerAG/gval"
	"github.com/PaesslerAG/jsonpath"
)

var jsonPathBuilder = gval.Full(jsonpath.PlaceholderExtension())

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
