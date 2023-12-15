package objects

import (
	"context"
	"path"
	"strings"

	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/common"
)

type uploadParams struct {
	Source      mgcSchemaPkg.FilePath `json:"src" jsonschema:"description=Source file path to be uploaded" mgc:"positional"`
	BucketName  common.BucketName     `json:"bucket" jsonschema:"description=Name of the bucket to upload to" mgc:"positional"`
	Destination mgcSchemaPkg.URI      `json:"dst,omitempty" jsonschema:"description=Full destination path in the bucket with desired filename,example=dir/file.txt" mgc:"positional"`
}

type uploadTemplateResult struct {
	File string `json:"file"`
	URI  string `json:"uri"`
}

var getUpload = utils.NewLazyLoader[core.Executor](func() core.Executor {
	executor := core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "upload",
			Description: "Upload a file to a bucket",
		},
		upload,
	)

	return core.NewExecuteResultOutputOptions(executor, func(exec core.Executor, result core.Result) string {
		return "template=Uploaded file {{.file}} to {{.uri}}\n"
	})
})

func upload(ctx context.Context, params uploadParams, cfg common.Config) (*uploadTemplateResult, error) {
	dst := params.BucketName.AsURI()
	dst = dst.JoinPath(params.Destination.String())
	fileName := path.Base(params.Source.String())
	if params.Destination.String() == "" || strings.HasSuffix(params.Destination.String(), "/") {
		// If it isn't a file path, don't rename, just append source with bucket URI
		dst = dst.JoinPath(fileName)
	}

	uploader, err := common.NewUploader(cfg, params.Source, dst)
	if err != nil {
		return nil, err
	}

	if err = uploader.Upload(ctx); err != nil {
		return nil, err
	}

	return &uploadTemplateResult{
		URI:  dst.String(),
		File: fileName,
	}, nil
}
