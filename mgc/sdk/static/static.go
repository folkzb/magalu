package static

import (
	"context"
	"fmt"

	"magalu.cloud/core"
)

type MyParams struct {
	SomeStringFlag string
	OtherIntFlag   int
}

type MyConfigs struct {
	SomeStringConfig string
}

type MyResult struct {
	SomeResultField string
}

func newStatic() *core.StaticExecute {
	return core.NewStaticExecuteSimple(
		"static",
		"34.56",
		"static first level",
		func(ctx context.Context, params MyParams, configs MyConfigs) (result *MyResult, err error) {
			fmt.Printf("TODO: static first level called. parameters=%q, configs=%q\n", params, configs)
			if root := core.GrouperFromContext(ctx); root != nil {
				_, _ = root.VisitChildren(func(child core.Descriptor) (run bool, err error) {
					println(">>> root child: ", child.Name())
					return true, nil
				})
			}
			if auth := core.AuthFromContext(ctx); auth != nil {
				println("I have auth from context", auth)
				return &MyResult{SomeResultField: "some value"}, nil
			}
			return &MyResult{SomeResultField: "some value"}, nil
		},
	)
}
