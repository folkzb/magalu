package config

import (
	"magalu.cloud/core"
)

func NewGroup() *core.StaticGroup {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:        "config",
			Description: "Config related commands",
		},
		[]core.Descriptor{
			newList(),
			newGet(),
			newSet(),
			newDelete(),
		},
	)
}
