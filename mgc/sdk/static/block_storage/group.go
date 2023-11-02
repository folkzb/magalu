package block_storage

import (
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
	attachment "magalu.cloud/sdk/static/block_storage/volume-attach"
)

var GetGroup = utils.NewLazyLoader[core.Grouper](newGroup)

func newGroup() core.Grouper {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:        "block-storage",
			Summary:     "Operations for Block Storage API",
			Description: `Create and manage Volumes via the Block Storage API`,
		},
		[]core.Descriptor{
			attachment.GetGroup(),
		},
	)
}
