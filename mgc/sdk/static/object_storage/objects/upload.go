package objects

import (
	"context"
	"path"

	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/common"
)

type uploadParams struct {
	Destination mgcSchemaPkg.URI      `json:"dst" jsonschema:"description=Full destination path in the bucket with s3 prefix,example=s3://bucket1" mgc:"positional"`
	Source      mgcSchemaPkg.FilePath `json:"src" jsonschema:"description=Source file path to be uploaded" mgc:"positional"`
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
	dst := params.Destination
	fileName := path.Base(params.Source.String())
	if isDirPath(dst.AsFilePath()) {
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
