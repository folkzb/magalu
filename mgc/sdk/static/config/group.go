package config

import (
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
)

var GetGroup = utils.NewLazyLoader[*core.StaticGroup](newGroup)

func newGroup() *core.StaticGroup {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:    "config",
			Summary: "Manage Configuration values",
			Description: `Configuration values are available to be set so that they persist between
different executions of the MgcSDK. They reside in a YAML file when set.
Config values may also be loaded via Environment Variables. Any Config available
(see 'list') may be exported as an env variable in uppercase with the 'MGC_' prefix`,
		},
		[]core.Descriptor{
			getList(),
			getGet(),
			getSet(),
			getDelete(),
		},
	)
}
