package object_storage

import (
	"magalu.cloud/core"
	"magalu.cloud/sdk/static/object_storage/buckets"
	"magalu.cloud/sdk/static/object_storage/objects"
)

func NewGroup() core.Grouper {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:    "object-storage",
			Version: "Operations for Object Storage API",
		},
		[]core.Descriptor{
			buckets.NewGroup(), // object-storage buckets
			objects.NewGroup(), // object-storage objects
		},
	)
}
