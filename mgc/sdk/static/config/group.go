package config

import (
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
)

var GetGroup = utils.NewLazyLoader[*core.StaticGroup](newGroup)

func newGroup() *core.StaticGroup {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:        "config",
			Description: "Config related commands",
		},
		[]core.Descriptor{
			getList(),
			getGet(),
			getSet(),
			getDelete(),
		},
	)
}
