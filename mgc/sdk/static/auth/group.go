package auth

import (
	"magalu.cloud/core"
	"magalu.cloud/sdk/static/auth/tenant"
)

func NewGroup() *core.StaticGroup {
	return core.NewStaticGroup(
		"auth",
		"",
		"",
		[]core.Descriptor{
			newLogin(),
			newAccessToken(),
			tenant.NewGroup(),
		},
	)
}
