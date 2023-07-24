package sdk

import "core"

func newStaticRoot() *core.StaticGroup {
	return core.NewStaticGroup(
		"Static Groups Root",
		"",
		"",
		[]core.Descriptor{
			newStaticRootStatic(), // cmd: "static"
			newStaticRootVpc(),    // cmd: "vpc"
		},
	)
}
