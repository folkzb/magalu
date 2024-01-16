package objects

import (
	"context"
	"fmt"
	"os"
	"path"

	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/common"
)

var getDownload = utils.NewLazyLoader[core.Executor](func() core.Executor {
	executor := core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "download",
			Description: "Download an object from a bucket",
		},
		download,
	)

	return core.NewExecuteResultOutputOptions(executor, func(exec core.Executor, result core.Result) string {
		return "template=Downloaded from {{.src}} to {{.dst}}\n"
	})
})

func downloadSingleFile(ctx context.Context, cfg common.Config, src mgcSchemaPkg.URI, dst mgcSchemaPkg.FilePath) error {
	req, err := common.NewDownloadRequest(ctx, cfg, src)
	if err != nil {
		return err
	}

	resp, err := common.SendRequest(ctx, req)
	if err != nil {
		return err
	}

	dir := path.Dir(dst.String())
	if len(dir) != 0 {
		if err := os.MkdirAll(dir, utils.DIR_PERMISSION); err != nil {
			return err
		}
	}

	if err := common.WriteToFile(ctx, resp.Body, resp.ContentLength, dst); err != nil {
		return err
	}

	return nil
}

func download(ctx context.Context, p common.DownloadObjectParams, cfg common.Config) (result core.Value, err error) {
	dst, err := common.GetDestination(p.Destination, p.Source)
	if err != nil {
		return nil, fmt.Errorf("no destination specified and could not use local dir: %w", err)
	}

	err = downloadSingleFile(ctx, cfg, p.Source, dst)

	if err != nil {
		return nil, err
	}

	return common.DownloadObjectParams{Source: p.Source, Destination: mgcSchemaPkg.FilePath(dst)}, nil
}
