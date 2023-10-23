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
			IsInternal:  false, // TODO: true
		},
		[]core.Descriptor{
			getJsonGroup(),    // http json
			getHttpExecutor(), // http do
		},
	)
})
