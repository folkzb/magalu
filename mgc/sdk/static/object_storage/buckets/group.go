package buckets

import (
	"magalu.cloud/core"
)

func NewGroup() core.Grouper {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:        "buckets",
			Description: "Bucket operations for Object Storage API",
		},
		[]core.Descriptor{
			getCreate(), // object-storage buckets create
			getDelete(), // object-storage buckets delete
			getList(),   // object-storage buckets list
		},
	)
}
