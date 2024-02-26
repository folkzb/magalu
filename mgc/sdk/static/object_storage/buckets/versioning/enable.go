package versioning

import (
	"context"
	"net/http"

	"magalu.cloud/core"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/common"
)

type enableBucketVersioningParams struct {
	Bucket common.BucketName `json:"bucket" jsonschema:"description=Bucket name to enable versioning" mgc:"positional"`
}

var getEnable = utils.NewLazyLoader(func() core.Executor {
	return core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "enable",
			Description: "Enable versioning for a Bucket",
		},
		enableBucketVersioning,
	)
})

func enableBucketVersioning(ctx context.Context, params enableBucketVersioningParams, cfg common.Config) (core.Value, error) {
	req, err := newEnableBucketVersioningRequest(ctx, params.Bucket, cfg)
	if err != nil {
		return nil, err
	}

	res, err := common.SendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	return common.UnwrapResponse[core.Value](res, req)
}

func newEnableBucketVersioningRequest(ctx context.Context, bucketName common.BucketName, cfg common.Config) (*http.Request, error) {
	return newSetBucketVersioningRequest(
		ctx,
		bucketName,
		cfg,
		versioningConfiguration{
			Status:    "Enabled",
			Namespace: "http://s3.amazonaws.com/doc/2006-03-01/",
		},
	)
}
