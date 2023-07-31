package port

import "magalu.cloud/core"

func NewGroup() *core.StaticGroup {
	return core.NewStaticGroup(
		"port",
		"",
		"",
		[]core.Descriptor{
			newStaticExample(), // cmd: vpc port static_example
		},
	)
}
