package profile

import (
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
)

var GetGroup = utils.NewLazyLoader[core.Grouper](func() core.Grouper {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name: "profile",
			Description: `Profiles hold auth and runtime configuration for the MgcSDK, like tokens and log filter settings.
Users can create as many profiles as you choose to. Auth and config operations will affect only the
current profile, so users can alter and switch between profiles without loosing the previous configuration`,
			Summary: "Profile related commands",
		},
		[]core.Descriptor{
			getCurrent(),
			getCreate(),
			getSetCurrent(),
		},
	)
})
