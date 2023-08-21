package tenant

import "magalu.cloud/core"

func NewGroup() core.Grouper {
	return core.NewStaticGroup(
		"tenant",
		"",
		"Tenant-related operations",
		[]core.Descriptor{
			newList(),
			newSelect(),
			newCurrent(),
		},
	)
}
