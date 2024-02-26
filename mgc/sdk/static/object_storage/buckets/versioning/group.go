package versioning

import (
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
)

var GetGroup = utils.NewLazyLoader(func() core.Grouper {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:        "versioning",
			Description: "Manage bucket versioning",
		},
		func() []core.Descriptor {
			return []core.Descriptor{
				getGet(),    // object-storage buckets versioning get
				getEnable(), // object-storage buckets versioning enable
			}
		},
	)
})
