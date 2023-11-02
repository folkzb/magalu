package attachment

import (
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
)

var GetGroup = utils.NewLazyLoader[core.Grouper](newGroup)

func newGroup() core.Grouper {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:        "volume-attachment",
			Summary:     "Block Storage Volume Attachment",
			Description: `Create and manage Volume Attachments`,
		},
		[]core.Descriptor{
			getCreate(),
			getGet(),
			getUpdate(),
			getDelete(),
		},
	)
}
