package buckets

import (
	"context"

	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/common"
)

type publicUrlParams struct {
	Destination mgcSchemaPkg.URI `json:"dst" jsonschema:"description=Path of the bucket to generate the public url,example=bucket1" mgc:"positional"`
}

var getPublicUrl = utils.NewLazyLoader[core.Executor](func() core.Executor {
	executor := core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "public-url",
			Description: "Get bucket public url",
		},
		bucketPublicUrl,
	)
	return core.NewExecuteResultOutputOptions(executor, func(exec core.Executor, result core.Result) string {
		return "template={{.url}}\n"
	})
})

func bucketPublicUrl(ctx context.Context, p publicUrlParams, cfg common.Config) (*common.PublicUrlResult, error) {
	return common.PublicUrl(ctx, cfg, p.Destination)
}
