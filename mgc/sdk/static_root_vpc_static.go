package sdk

import "core"

func newStaticRootVpcStatic() *core.StaticExecute {
	return core.NewStaticExecute(
		"static",
		"",
		"static second level",
		&core.Schema{},
		&core.Schema{},
		func(parameters, configs map[string]core.Value) (result core.Value, err error) {
			println("TODO: vpc static (second level) called")
			return nil, nil
		},
	)
}
