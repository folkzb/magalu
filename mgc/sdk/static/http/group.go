package http

import (
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
)

var GetGroup = utils.NewLazyLoader(func() core.Grouper {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:        "http",
			Description: "HTTP access",
			IsInternal:  utils.BoolPtr(true),
		},
		func() []core.Descriptor {
			return []core.Descriptor{
				getJsonGroup(),    // http json
				getHttpExecutor(), // http do
			}
		},
	)
})
