package sdk

import "magalu.cloud/core"

func newStaticRootVpc() *core.StaticGroup {
	return core.NewStaticGroup(
		"vpc",
		"",
		"",
		[]core.Descriptor{
			newStaticRootVpcStatic(), // cmd: vpc static
			newStaticRootVpcPort(),   // cmd: vpc port
		},
	)
}
