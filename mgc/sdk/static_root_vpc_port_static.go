package sdk

import "core"

func newStaticRootVpcPortStatic() *core.StaticExecute {
	return core.NewStaticExecute(
		"static",
		"",
		"static third level",
		&core.Schema{},
		&core.Schema{},
		func(parameters, configs map[string]core.Value) (result core.Value, err error) {
			println("TODO: vpc port static (third level) called")
			return nil, nil
		},
	)
}
