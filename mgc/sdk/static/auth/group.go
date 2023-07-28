package auth

import (
	"magalu.cloud/core"
)

func NewGroup() *core.StaticGroup {
	return core.NewStaticGroup(
		"auth",
		"",
		"",
		[]core.Descriptor{
			newLogin(),       // cmd: auth login
			newAccessToken(), // cmd: auth access_token
		},
	)
}
