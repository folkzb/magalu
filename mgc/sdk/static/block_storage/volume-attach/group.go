package attachment

import (
	"magalu.cloud/core"
)

func NewGroup() core.Grouper {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:        "volume-attach",
			Description: "Block Storage Volume Attachment",
		},
		[]core.Descriptor{
			newCreate(),
			newGet(),
			newUpdate(),
			newDelete(),
		},
	)
}
