package port

import (
	"context"

	"magalu.cloud/core"
)

func newStatic() *core.StaticExecute {
	return core.NewStaticExecuteSimple(
		"static",
		"",
		"static third level",
		func(ctx context.Context) (result core.Value, err error) {
			println("TODO: vpc port static (third level) called")
			return nil, nil
		},
	)
}
