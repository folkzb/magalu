package objects

import (
	"context"
	"fmt"
	"math"
	"os"
	"path"

	"go.uber.org/zap"
	"magalu.cloud/core"
	"magalu.cloud/core/pipeline"
	"magalu.cloud/core/progress_report"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/common"
)

var downloadAllObjectsLogger *zap.SugaredLogger

func downloadAllLogger() *zap.SugaredLogger {
	if downloadAllObjectsLogger == nil {
		downloadAllObjectsLogger = logger().Named("download")
	}
	return downloadAllObjectsLogger
}

type downloadAllObjectsParams struct {
	Source         mgcSchemaPkg.URI      `json:"src" jsonschema:"description=Path of objects to be downloaded,example=s3://mybucket" mgc:"positional"`
	Destination    mgcSchemaPkg.FilePath `json:"dst,omitempty" jsonschema:"description=Path to save files,example=path/to/folder" mgc:"positional"`
	common.Filters `json:",squash"`      // nolint
}

var getDownloadAll = utils.NewLazyLoader[core.Executor](func() core.Executor {
	executor := core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "download-all",
			Description: "Download all objects from a bucket",
		},
		downloadAll,
	)

	return core.NewExecuteResultOutputOptions(executor, func(exec core.Executor, result core.Result) string {
		return "template=Downloaded from {{.src}} to {{.dst}}\n"
	})
})

func createObjectDownloadProcessor(cfg common.Config, params downloadAllObjectsParams, reportChan chan<- downloadAllProgressReport) pipeline.Processor[pipeline.WalkDirEntry, error] {
	return func(ctx context.Context, dirEntry pipeline.WalkDirEntry) (error, pipeline.ProcessStatus) {
		bucketName := common.NewBucketNameFromURI(params.Source)
		rootURI := bucketName.AsURI()
		var err error

		defer func() {
			reportChan <- downloadAllProgressReport{uint64(1), err}
		}()

		objURI := rootURI.JoinPath(dirEntry.Path())

		if dirEntry.Err() != nil {
			err = &common.ObjectError{Url: mgcSchemaPkg.URI(objURI), Err: dirEntry.Err()}
			return err, pipeline.ProcessSkip
		}

		_, ok := dirEntry.DirEntry().(*common.BucketContent)
		if !ok {
			err = &common.ObjectError{Url: mgcSchemaPkg.URI(objURI), Err: fmt.Errorf("expected object, got directory")}
			return err, pipeline.ProcessSkip
		}

		downloadAllLogger().Infow("Downloading object", "uri", objURI)
		req, err := common.NewDownloadRequest(ctx, cfg, mgcSchemaPkg.URI(objURI))
		if err != nil {
			err = &common.ObjectError{Url: mgcSchemaPkg.URI(objURI), Err: err}
			return err, pipeline.ProcessSkip
		}

		resp, err := common.SendRequest(ctx, req)
		if err != nil {
			err = &common.ObjectError{Url: mgcSchemaPkg.URI(objURI), Err: err}
			return err, pipeline.ProcessSkip
		}

		dir := path.Dir(dirEntry.Path())
		if err = os.MkdirAll(path.Join(params.Destination.String(), dir), utils.DIR_PERMISSION); err != nil {
			err = &common.ObjectError{Url: mgcSchemaPkg.URI(objURI), Err: err}
			return err, pipeline.ProcessSkip
		}

		if err = common.WriteToFile(ctx, resp.Body, resp.ContentLength, params.Destination.Join(dirEntry.Path())); err != nil {
			err = &common.ObjectError{Url: mgcSchemaPkg.URI(objURI), Err: err}
			return err, pipeline.ProcessSkip
		}

		return nil, pipeline.ProcessOutput
	}
}

type downloadAllProgressReport struct {
	files uint64
	err   error
}

func downloadMultipleFiles(ctx context.Context, cfg common.Config, params downloadAllObjectsParams) error {
	listParams := common.ListObjectsParams{
		Destination: params.Source,
		Recursive:   true,
		PaginationParams: common.PaginationParams{
			MaxItems: math.MaxInt64,
		},
	}

	objs := common.ListGenerator(ctx, listParams, cfg)
	objs = common.ApplyFilters(ctx, objs, params.FilterParams, nil)

	reportProgress := progress_report.FromContext(ctx)
	reportChan := make(chan downloadAllProgressReport)
	defer close(reportChan)

	go reportDownloadAllProgress(reportProgress, reportChan, params)

	downloadObjectsErrorChan := pipeline.ParallelProcess(ctx, cfg.Workers, objs, createObjectDownloadProcessor(cfg, params, reportChan), nil)
	downloadObjectsErrorChan = pipeline.Filter(ctx, downloadObjectsErrorChan, pipeline.FilterNonNil[error]{})

	objErr, _ := pipeline.SliceItemConsumer[utils.MultiError](ctx, downloadObjectsErrorChan)
	if len(objErr) > 0 {
		return objErr
	}

	return nil
}

func reportDownloadAllProgress(reportProgress progress_report.ReportProgress, reportChan <-chan downloadAllProgressReport, params downloadAllObjectsParams) {
	reportMsg := "downloading objects from bucket: " + params.Source.String()
	total := uint64(1)
	progress := uint64(0)

	// total here must be reported as one, otherwise the progress-bar shows
	// an animation we do not wish the user to see
	reportProgress(reportMsg, progress, total, progress_report.UnitsNone, nil)

	var errors utils.MultiError
	for report := range reportChan {
		progress += report.files
		total += report.files

		if report.err != nil {
			errors = append(errors, report.err)
		}

		reportProgress(reportMsg, progress, total, progress_report.UnitsNone, nil)
	}

	if len(errors) > 0 {
		reportProgress(reportMsg, progress, total, progress_report.UnitsNone, errors)
		return
	}

	reportProgress(reportMsg, total, total, progress_report.UnitsNone, progress_report.ErrorProgressDone)
}

func downloadAll(ctx context.Context, p downloadAllObjectsParams, cfg common.Config) (result core.Value, err error) {
	dst, err := common.GetDestination(p.Destination, p.Source)
	if err != nil {
		return nil, fmt.Errorf("no destination specified and could not use local dir: %w", err)
	}
	p.Destination = dst
	err = downloadMultipleFiles(ctx, cfg, p)

	if err != nil {
		return nil, err
	}

	return common.DownloadObjectParams{Source: p.Source, Destination: mgcSchemaPkg.FilePath(dst)}, nil
}
