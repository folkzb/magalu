package static

import (
	"magalu.cloud/core"
	"magalu.cloud/sdk/static/auth"
	"magalu.cloud/sdk/static/block_storage"
	"magalu.cloud/sdk/static/config"
	"magalu.cloud/sdk/static/object_storage"
)

func NewGroup() *core.StaticGroup {
	return core.NewStaticGroup(
		core.DescriptorSpec{Name: "Static Groups Root"},
		[]core.Descriptor{
			auth.GetGroup(),           // cmd: "auth"
			config.GetGroup(),         // cmd: "config"
			object_storage.GetGroup(), // cmd: "object-storage"
			block_storage.GetGroup(),  // cmd: "block-storage"
		},
	)
}
