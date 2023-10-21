package tenant

import "magalu.cloud/core"

func NewGroup() core.Grouper {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:        "tenant",
			Description: "Tenant-related operations",
		},
		[]core.Descriptor{
			newList(),
			newSelect(),
			newCurrent(),
		},
	)
}
