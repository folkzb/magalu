package common

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"

	"go.uber.org/zap"
	"magalu.cloud/core"
	"magalu.cloud/core/pipeline"
	"magalu.cloud/core/progress_report"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
)

var copyAllObjectsLogger *zap.SugaredLogger

func copyAllLogger() *zap.SugaredLogger {
	if copyAllObjectsLogger == nil {
		copyAllObjectsLogger = logger().Named("copy")
	}
	return copyAllObjectsLogger
}

type CopyObjectParams struct {
	Source      mgcSchemaPkg.URI `json:"src" jsonschema:"description=Path of the object in a bucket to be copied,example=bucket1/file.txt" mgc:"positional"`
	Destination mgcSchemaPkg.URI `json:"dst" jsonschema:"description=Full destination path in the bucket with desired filename,example=bucket2/dir/file.txt" mgc:"positional"`
}

type CopyAllObjectsParams struct {
	Source      mgcSchemaPkg.URI `json:"src" jsonschema:"description=Path of objects in a bucket to be copied,example=bucket1" mgc:"positional"`
	Destination mgcSchemaPkg.URI `json:"dst" jsonschema:"description=Full destination path in the bucket,example=bucket2/dir/" mgc:"positional"`
	Filters     `json:",squash"` // nolint
}

type copyAllProgressReport struct {
	files uint64
	total uint64
	err   error
}

func newCopyRequest(ctx context.Context, cfg Config, src mgcSchemaPkg.URI, dst mgcSchemaPkg.URI) (*http.Request, error) {
	host, err := BuildBucketHostWithPath(cfg, NewBucketNameFromURI(dst), dst.Path())
	if err != nil {
		return nil, core.UsageError{Err: err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, string(host), nil)
	if err != nil {
		return nil, err
	}

	copySource, err := url.JoinPath(src.Hostname(), src.Path())
	if err != nil {
		return nil, core.UsageError{Err: fmt.Errorf("badly specified source URI: %w", err)}
	}

	req.Header.Set("x-amz-copy-source", copySource)

	return req, nil
}

func createObjectCopyProcessor(cfg Config, params CopyAllObjectsParams, reportChan chan<- copyAllProgressReport) pipeline.Processor[pipeline.WalkDirEntry, error] {
	return func(ctx context.Context, dirEntry pipeline.WalkDirEntry) (error, pipeline.ProcessStatus) {
		bucketName := NewBucketNameFromURI(params.Source)
		rootURI := bucketName.AsURI()
		var err error

		defer func() {
			reportChan <- copyAllProgressReport{uint64(1), 0, err}
		}()

		objURI := rootURI.JoinPath(dirEntry.Path())

		if dirEntry.Err() != nil {
			err = &ObjectError{Url: mgcSchemaPkg.URI(objURI), Err: dirEntry.Err()}
			return err, pipeline.ProcessOutput
		}

		_, ok := dirEntry.DirEntry().(*BucketContent)
		if !ok {
			err = &ObjectError{Url: mgcSchemaPkg.URI(objURI), Err: fmt.Errorf("expected object, got directory")}
			return err, pipeline.ProcessOutput
		}

		copyAllLogger().Infow("Copying object", "uri", objURI)
		err = CopySingleFile(ctx, cfg, objURI, params.Destination.JoinPath(dirEntry.Path()))
		if err != nil {
			return err, pipeline.ProcessAbort
		}

		if err != nil {
			err = &ObjectError{Url: mgcSchemaPkg.URI(objURI), Err: err}
			return err, pipeline.ProcessOutput
		}

		return nil, pipeline.ProcessOutput
	}
}

func CopyMultipleFiles(ctx context.Context, cfg Config, params CopyAllObjectsParams) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	listParams := ListObjectsParams{
		Destination: params.Source,
		Recursive:   true,
		PaginationParams: PaginationParams{
			MaxItems: math.MaxInt64,
		},
	}

	reportProgress := progress_report.FromContext(ctx)
	reportChan := make(chan copyAllProgressReport)
	defer close(reportChan)

	onNewPage := func(objCount uint64) {
		reportChan <- copyAllProgressReport{0, objCount, nil}
	}

	objs := ListGenerator(ctx, listParams, cfg, onNewPage)
	objs = ApplyFilters(ctx, objs, params.FilterParams, cancel)

	go reportCopyAllProgress(reportProgress, reportChan, params)

	copyObjectsErrorChan := pipeline.ParallelProcess(ctx, cfg.Workers, objs, createObjectCopyProcessor(cfg, params, reportChan), nil)
	copyObjectsErrorChan = pipeline.Filter(ctx, copyObjectsErrorChan, pipeline.FilterNonNil[error]{})

	objErr, err := pipeline.SliceItemConsumer[utils.MultiError](ctx, copyObjectsErrorChan)
	if err != nil {
		return err
	}
	if len(objErr) > 0 {
		return objErr
	}

	return nil
}

func CopySingleFile(ctx context.Context, cfg Config, src mgcSchemaPkg.URI, dst mgcSchemaPkg.URI) error {
	reportProgress := progress_report.FromContext(ctx)
	reportMsg := "Copying object from " + src.String() + " to " + dst.String()
	progress := uint64(0)
	total := uint64(1)

	if dst.IsRoot() {
		dst = dst.JoinPath(src.Filename())
	}

	reportProgress(reportMsg, progress, progress, progress_report.UnitsNone, nil)

	req, err := newCopyRequest(ctx, cfg, src, dst)
	if err != nil {
		reportProgress(reportMsg, progress, progress, progress_report.UnitsNone, err)
		return err
	}

	resp, err := SendRequest(ctx, req)
	if err != nil {
		reportProgress(reportMsg, progress, progress, progress_report.UnitsNone, err)
		return err
	}

	reportProgress(reportMsg, total, total, progress_report.UnitsNone, progress_report.ErrorProgressDone)

	return ExtractErr(resp, req)
}

func reportCopyAllProgress(reportProgress progress_report.ReportProgress, reportChan <-chan copyAllProgressReport, params CopyAllObjectsParams) {
	reportMsg := "copying objects from bucket: " + params.Source.String()
	total := uint64(1)
	progress := uint64(0)

	// total here must be reported as one, otherwise the progress-bar shows
	// an animation we do not wish the user to see
	reportProgress(reportMsg, progress, total, progress_report.UnitsNone, nil)

	var errors utils.MultiError
	for report := range reportChan {
		progress += report.files
		total += report.total

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
