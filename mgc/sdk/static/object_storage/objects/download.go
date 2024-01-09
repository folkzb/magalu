package objects

import (
	"context"
	"fmt"
	"os"
	"path"

	"magalu.cloud/core"
	"magalu.cloud/core/progress_report"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/common"
)

var getDownload = utils.NewLazyLoader[core.Executor](func() core.Executor {
	executor := core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "download",
			Description: "download an object from a bucket",
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

	if err := common.WriteToFile(resp.Body, dst); err != nil {
		return err
	}

	return nil
}

func download(ctx context.Context, p common.DownloadObjectParams, cfg common.Config) (result core.Value, err error) {
	dst, err := common.GetDestination(p.Destination, p.Source)
	if err != nil {
		return nil, fmt.Errorf("no destination specified and could not use local dir: %w", err)
	}

	reportProgress := progress_report.FromContext(ctx)
	total := uint64(1)
	progress := uint64(0)
	name := p.Source.String()

	reportProgress(name, progress, total, progress_report.UnitsNone, nil)

	err = downloadSingleFile(ctx, cfg, p.Source, dst)

	if err != nil {
		reportProgress(name, progress, total, progress_report.UnitsNone, err)
		return nil, err
	}

	reportProgress(name, progress+1, total, progress_report.UnitsNone, progress_report.ErrorProgressDone)

	return common.DownloadObjectParams{Source: p.Source, Destination: mgcSchemaPkg.FilePath(dst)}, nil
}
