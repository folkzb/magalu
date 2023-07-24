package port

import "magalu.cloud/core"

func NewGroup() *core.StaticGroup {
	return core.NewStaticGroup(
		"port",
		"",
		"",
		[]core.Descriptor{
			newStatic(), // cmd: vpc port static
		},
	)
}
