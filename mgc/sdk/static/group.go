package static

import (
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/auth"
	"magalu.cloud/sdk/static/block_storage"
	"magalu.cloud/sdk/static/config"
	"magalu.cloud/sdk/static/http"
	"magalu.cloud/sdk/static/object_storage"
	"magalu.cloud/sdk/static/profile"
)

var GetGroup = utils.NewLazyLoader(func() core.Grouper {
	return core.NewStaticGroup(
		core.DescriptorSpec{Name: "Static Groups Root"},
		func() []core.Descriptor {
			return []core.Descriptor{
				auth.GetGroup(),           // cmd: "auth"
				config.GetGroup(),         // cmd: "config"
				object_storage.GetGroup(), // cmd: "object-storage"
				block_storage.GetGroup(),  // cmd: "block-storage"
				profile.GetGroup(),        // cmd: "profile"
				http.GetGroup(),           // cmd: "http"
			}
		},
	)
})
