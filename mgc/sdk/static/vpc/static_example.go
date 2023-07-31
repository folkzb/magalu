package vpc

import (
	"context"

	"magalu.cloud/core"
)

func newStaticExample() *core.StaticExecute {
	return core.NewStaticExecuteSimple(
		"static_example",
		"",
		"static second level",
		func(ctx context.Context) (result core.Value, err error) {
			println("TODO: vpc static_example (second level) called")
			return nil, nil
		},
	)
}
