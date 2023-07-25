package port

import (
	"context"

	"magalu.cloud/core"
)

func newStatic() *core.StaticExecute {
	return core.NewStaticExecute(
		"static",
		"",
		"static third level",
		&core.Schema{},
		&core.Schema{},
		func(ctx context.Context, parameters, configs map[string]core.Value) (result core.Value, err error) {
			println("TODO: vpc port static (third level) called")
			return nil, nil
		},
	)
}
