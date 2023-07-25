package static

import (
	"context"

	"magalu.cloud/core"
)

func newStatic() *core.StaticExecute {
	return core.NewStaticExecute(
		"static",
		"34.56",
		"static first level",
		// NOTE: these can (should?) be defined in JSON and unmarshal from string
		core.NewObjectSchema(
			map[string]*core.Schema{
				"param1": core.SetDescription(
					core.NewNumberSchema(),
					"Example static parameter of type number",
				),
			},
			[]string{},
		),
		&core.Schema{},
		func(ctx context.Context, parameters, configs map[string]core.Value) (result core.Value, err error) {
			println("TODO: static first level called")
			if root := core.GrouperFromContext(ctx); root != nil {
				_, _ = root.VisitChildren(func(child core.Descriptor) (run bool, err error) {
					println(">>> root child: ", child.Name())
					return true, nil
				})
			}
			if auth := core.AuthFromContext(ctx); auth != nil {
				println("I have auth from context", auth)
				return nil, nil
			}
			return nil, nil
		},
	)
}
