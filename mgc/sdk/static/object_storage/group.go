package object_storage

import (
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/buckets"
	"magalu.cloud/sdk/static/object_storage/objects"
)

var GetGroup = utils.NewLazyLoader[core.Grouper](newGroup)

func newGroup() core.Grouper {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:    "object-storage",
			Version: "Operations for Object Storage API",
		},
		[]core.Descriptor{
			buckets.GetGroup(), // object-storage buckets
			objects.GetGroup(), // object-storage objects
		},
	)
}
