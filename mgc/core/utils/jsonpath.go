package utils

import (
	"context"

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
