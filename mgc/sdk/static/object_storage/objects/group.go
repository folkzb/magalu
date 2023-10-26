package objects

import (
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
)

var GetGroup = utils.NewLazyLoader[core.Grouper](newGroup)

func newGroup() core.Grouper {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:        "objects",
			Description: "Object operations for Object Storage API",
		},
		[]core.Descriptor{
			getDelete(),   // object-storage objects delete
			getDownload(), // object-storage objects download
			getList(),     // object-storage objects list
			getUpload(),   // object-storage objects upload
		},
	)
}
