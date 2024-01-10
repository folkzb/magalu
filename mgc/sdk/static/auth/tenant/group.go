package tenant

import (
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
)

var GetGroup = utils.NewLazyLoader(func() core.Grouper {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:    "tenant",
			Summary: "Manage Tenants",
			Description: `Tenants work like sub-accounts. You may have more than one Tenant under your
Magalu Cloud account and they each store their data separately, but are billed
under the same account`,
		},
		func() []core.Descriptor {
			return []core.Descriptor{
				getList(),
				getSelect(),
				getCurrent(),
			}
		},
	)
})
