package objects

import (
	"context"

	"magalu.cloud/core"
	"magalu.cloud/core/pipeline"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/common"
)

type listResponse struct {
	Contents       []*common.BucketContent `xml:"Contents"`
	CommonPrefixes []*common.Prefix        `xml:"CommonPrefixes"`
}

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

func List(ctx context.Context, params common.ListObjectsParams, cfg common.Config) (result listResponse, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	objChan := common.ListGenerator(ctx, params, cfg)

	entries, err := pipeline.SliceItemLimitedConsumer[[]pipeline.WalkDirEntry](ctx, params.MaxItems, objChan)
	if err != nil {
		return result, err
	}

	contents := make([]*common.BucketContent, 0, len(entries))
	commonPrefixes := make([]*common.Prefix, 0)
	for _, entry := range entries {
		if entry.Err() != nil {
			return result, entry.Err()
		}
		if entry.DirEntry().IsDir() {
			commonPrefixes = append(commonPrefixes, entry.DirEntry().(*common.Prefix))
		} else {
			contents = append(contents, entry.DirEntry().(*common.BucketContent))
		}
	}

	result = listResponse{
		Contents:       contents,
		CommonPrefixes: commonPrefixes,
	}
	return result, nil
}
