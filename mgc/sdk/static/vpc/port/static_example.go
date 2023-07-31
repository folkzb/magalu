package port

import (
	"context"

	"magalu.cloud/core"
)

func newStaticExample() *core.StaticExecute {
	return core.NewStaticExecuteSimple(
		"static_example",
		"",
		"static third level",
		func(ctx context.Context) (result core.Value, err error) {
			println("TODO: vpc port static_example (third level) called")
			return nil, nil
		},
	)
}
