package tenant

import (
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
)

var GetGroup = utils.NewLazyLoader[core.Grouper](newGroup)

func newGroup() core.Grouper {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:        "tenant",
			Description: "Tenant-related operations",
		},
		[]core.Descriptor{
			getList(),
			getSelect(),
			getCurrent(),
		},
	)
}
