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
	Source              mgcSchemaPkg.URI      `json:"src" jsonschema:"description=Path of objects to be downloaded,example=s3://mybucket/" mgc:"positional"`
	Destination         mgcSchemaPkg.FilePath `json:"dst,omitempty" jsonschema:"description=Path to save files,example=path/to/folder" mgc:"positional"`
	common.FilterParams `json:",squash"`      // nolint
}

var getDownloadAll = utils.NewLazyLoader[core.Executor](func() core.Executor {
	executor := core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "download-all",
			Description: "download all objects from a bucket",
		},
		downloadAll,
	)

	return core.NewExecuteResultOutputOptions(executor, func(exec core.Executor, result core.Result) string {
		return "template=Downloaded from {{.src}} to {{.dst}}\n"
	})
})

func downloadMultipleFiles(ctx context.Context, cfg common.Config, params downloadAllObjectsParams) error {
	src := params.Source
	dst := params.Destination
	listParams := common.ListObjectsParams{
		Destination: src,
		Recursive:   true,
		PaginationParams: common.PaginationParams{
			MaxItems: math.MaxInt64,
		},
	}

	objs := common.ListGenerator(ctx, listParams, cfg)

	if params.Include != "" {
		includeFilter := pipeline.FilterRuleIncludeOnly[pipeline.WalkDirEntry]{
			Pattern: pipeline.FilterWalkDirEntryIncludeGlobMatch{Pattern: params.Include},
		}

		objs = pipeline.Filter[pipeline.WalkDirEntry](ctx, objs, includeFilter)
	}

	if params.Exclude != "" {
		excludeFilter := pipeline.FilterRuleNot[pipeline.WalkDirEntry]{
			Not: pipeline.FilterWalkDirEntryIncludeGlobMatch{Pattern: params.Exclude},
		}
		objs = pipeline.Filter[pipeline.WalkDirEntry](ctx, objs, excludeFilter)
	}

	entries, err := pipeline.SliceItemConsumer[[]pipeline.WalkDirEntry](ctx, objs)
	if err != nil {
		return err
	}

	bucketName := common.NewBucketNameFromURI(src)
	rootURI := bucketName.AsURI()

	reportProgress := progress_report.FromContext(ctx)
	reportMsg := "downloading objects from bucket: " + params.Source.String()
	progress := uint64(0)
	total := uint64(len(entries))

	reportProgress(reportMsg, progress, total, progress_report.UnitsNone, nil)

	var errors utils.MultiError
	for _, entry := range entries {
		objURI := rootURI.JoinPath(entry.Path())

		if err := entry.Err(); err != nil {
			errors = append(errors, &common.ObjectError{Url: objURI, Err: err})
			continue
		}

		_, ok := entry.DirEntry().(*common.BucketContent)
		if !ok {
			errors = append(errors, &common.ObjectError{Url: objURI, Err: fmt.Errorf("expected object, got directory")})
			continue
		}

		downloadAllLogger().Infow("Downloading object", "uri", objURI)
		// TODO: change API to use BucketName, URI and FilePath
		req, err := common.NewDownloadRequest(ctx, cfg, mgcSchemaPkg.URI(objURI))
		if err != nil {
			errors = append(errors, &common.ObjectError{Url: objURI, Err: err})
			continue
		}

		resp, err := common.SendRequest(ctx, req)
		if err != nil {
			errors = append(errors, &common.ObjectError{Url: objURI, Err: err})
			continue
		}

		dir := path.Dir(entry.Path())
		if err := os.MkdirAll(path.Join(dst.String(), dir), utils.DIR_PERMISSION); err != nil {
			errors = append(errors, &common.ObjectError{Url: objURI, Err: err})
			continue
		}

		if err := common.WriteToFile(resp.Body, dst.Join(entry.Path())); err != nil {
			errors = append(errors, &common.ObjectError{Url: objURI, Err: err})
			continue
		}

		progress += 1
		reportProgress(reportMsg, progress, total, progress_report.UnitsNone, nil)
	}

	reportProgress(reportMsg, total, total, progress_report.UnitsNone, progress_report.ErrorProgressDone)

	if len(errors) > 0 {
		return errors
	}

	return nil
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

	// TODO: change API to use BucketName, URI and FilePath
	return common.DownloadObjectParams{Source: p.Source, Destination: mgcSchemaPkg.FilePath(dst)}, nil
}
