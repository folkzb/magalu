package objects

import (
	"context"

	"magalu.cloud/core"
	"magalu.cloud/core/pipeline"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/common"
)

var getList = utils.NewLazyLoader[core.Executor](newList)

func newList() core.Executor {
	return core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "list",
			Description: "List all objects from a bucket",
		},
		List,
	)
}

func List(ctx context.Context, params common.ListObjectsParams, cfg common.Config) (result common.ListObjectsResponse, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	objChan := common.ListGenerator(ctx, params, cfg)

	entries, err := pipeline.SliceItemLimitedConsumer[[]common.BucketContentDirEntry](ctx, params.MaxItems, objChan)
	if err != nil {
		return result, err
	}

	contents := make([]*common.BucketContent, 0, len(entries))
	for _, entry := range entries {
		if entry.Err() != nil {
			return result, entry.Err()
		}

		contents = append(contents, entry.Object)
	}

	result = common.ListObjectsResponse{
		Contents: contents,
	}
	return result, nil
}
