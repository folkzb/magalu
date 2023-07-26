package vpc

import (
	"context"

	"magalu.cloud/core"
)

func newStatic() *core.StaticExecute {
	return core.NewStaticExecuteSimple(
		"static",
		"",
		"static second level",
		func(ctx context.Context) (result core.Value, err error) {
			println("TODO: vpc static (second level) called")
			return nil, nil
		},
	)
}
