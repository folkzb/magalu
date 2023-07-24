package sdk

import "core"

func newStaticRootStatic() *core.StaticExecute {
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
		func(parameters, configs map[string]core.Value) (result core.Value, err error) {
			println("TODO: static first level called")
			return nil, nil
		},
	)
}
