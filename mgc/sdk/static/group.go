package static

import (
	"magalu.cloud/core"
	"magalu.cloud/sdk/static/vpc"
)

func NewGroup() *core.StaticGroup {
	return core.NewStaticGroup(
		"Static Groups Root",
		"",
		"",
		[]core.Descriptor{
			newStatic(),    // cmd: "static"
			vpc.NewGroup(), // cmd: "vpc"
		},
	)
}
