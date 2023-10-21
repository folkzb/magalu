package block_storage

import (
	"magalu.cloud/core"
	attachment "magalu.cloud/sdk/static/block_storage/volume-attach"
)

func NewGroup() core.Grouper {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:        "block-storage",
			Description: "Operations for Block Storage API",
		},
		[]core.Descriptor{
			attachment.NewGroup(),
		},
	)
}
