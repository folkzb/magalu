package objectstorage

import (
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
)

var GetGroup = utils.NewLazyLoader(newGroup)

func newGroup() core.Grouper {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:        "object-storage",
			Description: "Credentials used for object storage",
		},
		[]core.Descriptor{
			getSet(),
		},
	)
}
