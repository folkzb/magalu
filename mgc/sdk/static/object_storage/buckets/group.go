package buckets

import (
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/buckets/acl"
	"magalu.cloud/sdk/static/object_storage/buckets/label"
	object_lock "magalu.cloud/sdk/static/object_storage/buckets/object-lock"
	"magalu.cloud/sdk/static/object_storage/buckets/policy"
	"magalu.cloud/sdk/static/object_storage/buckets/versioning"
)

var GetGroup = utils.NewLazyLoader[core.Grouper](func() core.Grouper {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:        "buckets",
			Description: "Bucket operations for Object Storage API",
		},
		func() []core.Descriptor {
			return []core.Descriptor{
				getCreate(),            // object-storage buckets create
				getDelete(),            // object-storage buckets delete
				getList(),              // object-storage buckets list
				getBucket(),            // object-storage buckets get
				getPublicUrl(),         // object-storage objects public-url
				acl.GetGroup(),         // object-storage buckets acl
				versioning.GetGroup(),  // object-storage buckets versioning
				policy.GetGroup(),      // object-storage buckets policy
				label.GetGroup(),       // object-storage buckets label
				object_lock.GetGroup(), // object-storage buckets object-lock
			}
		},
	)
})
