package objects

import (
	"context"
	"fmt"
	"net"
	"time"
	"strings"

	"github.com/MagaluCloud/magalu/mgc/core"
	mgcSchemaPkg "github.com/MagaluCloud/magalu/mgc/core/schema"
	"github.com/MagaluCloud/magalu/mgc/core/utils"
	"github.com/MagaluCloud/magalu/mgc/sdk/static/object_storage/common"
)

type uploadParams struct {
	Source       mgcSchemaPkg.FilePath `json:"src" jsonschema:"description=Source file path to be uploaded,example=./file.txt" mgc:"positional"`
	Destination  mgcSchemaPkg.URI      `json:"dst" jsonschema:"description=Full destination path in the bucket with desired filename,example=my-bucket/dir/file.txt" mgc:"positional"`
	StorageClass string                `json:"storage_class,omitempty" jsonschema:"description=Type of Storage in which to store object,example=cold,enum=,enum=standard,enum=cold,enum=glacier_ir,enum=cold_instant,default="`
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
	fullDstPath := params.Destination
	if fullDstPath == "" {
		return nil, core.UsageError{Err: fmt.Errorf("destination cannot be empty")}
	}

	srcPath := params.Source.AsURI().String()
	fileName := common.ExtractFileName(srcPath)

	if params.Destination.IsRoot() || strings.HasSuffix(fullDstPath.String(), "/") {
		fullDstPath = fullDstPath.JoinPath(fileName)
	}

	uploader, err := common.NewUploader(cfg, params.Source, fullDstPath, params.StorageClass)
	if err != nil {
		return nil, err
	}

	retries := cfg.Retries
	if retries <= 0 {
		retries = 0
	}
	backoff := 500 * time.Millisecond

	for i := 0; i <= retries; i++ {
		err = uploader.Upload(ctx)
		if err == nil {
			break
		}

		if isTemporaryErr(err) && i < retries {
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		return nil, err
	}

	return &uploadTemplateResult{
		URI:  fullDstPath.String(),
		File: fileName,
	}, nil
}

func isTemporaryErr(err error) bool {
	if err == nil {
		return false
	}

	if netErr, ok := err.(net.Error); ok {
		return netErr.Temporary() || netErr.Timeout()
	}

	errStr := err.Error()
	if strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "server misbehaving") ||
		strings.Contains(errStr, "dial tcp") ||
		strings.Contains(errStr, "i/o timeout") {
		return true
	}

	return false
}
