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
			newStatic(),     // cmd: vpc static
			port.NewGroup(), // cmd: vpc port
		},
	)
}
