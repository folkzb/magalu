package vpc

import (
	"magalu.cloud/core"
	"magalu.cloud/sdk/static/vpc/port"
)

func NewGroup() *core.StaticGroup {
	return core.NewStaticGroup(
		"vpc",
		"",
		"",
		[]core.Descriptor{
			newStaticExample(), // cmd: vpc static_example
			port.NewGroup(),    // cmd: vpc port
		},
	)
}
