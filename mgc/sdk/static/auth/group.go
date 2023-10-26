package auth

import (
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/auth/tenant"
)

var GetGroup = utils.NewLazyLoader[core.Grouper](newGroup)

func newGroup() core.Grouper {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:        "auth",
			Description: "Actions with ID Magalu to login, refresh tokens, change tenants and others",
		},
		[]core.Descriptor{
			getSet(),
			getLogin(),
			getAccessToken(),
			tenant.GetGroup(),
		},
	)
}
