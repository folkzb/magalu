package attachment

import (
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
)

var GetGroup = utils.NewLazyLoader[core.Grouper](newGroup)

func newGroup() core.Grouper {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:        "volume-attach",
			Description: "Block Storage Volume Attachment",
		},
		[]core.Descriptor{
			getCreate(),
			getGet(),
			getUpdate(),
			getDelete(),
		},
	)
}
