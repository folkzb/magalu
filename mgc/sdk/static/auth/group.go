package auth

import (
	"magalu.cloud/core"
	"magalu.cloud/sdk/static/auth/tenant"
)

func NewGroup() *core.StaticGroup {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:        "auth",
			Description: "Actions with ID Magalu to login, refresh tokens, change tenants and others",
		},
		[]core.Descriptor{
			newSet(),
			newLogin(),
			newAccessToken(),
			tenant.NewGroup(),
		},
	)
}
