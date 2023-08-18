package config

import (
	"magalu.cloud/core"
)

func NewGroup() *core.StaticGroup {
	return core.NewStaticGroup(
		"config",
		"",
		"Config related commands",
		[]core.Descriptor{
			newList(),
			newGet(),
			newSet(),
			newDelete(),
		},
	)
}
