package objects

import (
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
)

var GetGroup = utils.NewLazyLoader[core.Grouper](func() core.Grouper {
	return core.NewStaticGroup(
		core.DescriptorSpec{
			Name:        "objects",
			Description: "Object operations for Object Storage API",
		},
		func() []core.Descriptor {
			return []core.Descriptor{
				getDelete(),      // object-storage objects delete
				getDeleteAll(),   // object-storage objects delete-all
				getDownload(),    // object-storage objects download
				getDownloadAll(), // object-storage objects download-all
				getList(),        // object-storage objects list
				getUpload(),      // object-storage objects upload
				getUploadDir(),   // object-storage objects upload-dir
			}
		},
	)
})
