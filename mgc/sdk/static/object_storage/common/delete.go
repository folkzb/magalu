package common

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"path"
	"strings"

	"go.uber.org/zap"
	"magalu.cloud/core"
	"magalu.cloud/core/pipeline"
)

var deleteObjectsLogger *zap.SugaredLogger

func deleteLogger() *zap.SugaredLogger {
	if deleteObjectsLogger == nil {
		deleteObjectsLogger = logger().Named("delete")
	}
	return deleteObjectsLogger
}

type DeleteObjectParams struct {
	Destination string `json:"dst,omitempty" jsonschema:"description=Path of the object to be deleted,example=s3://bucket1/file1"`
}

type DeleteAllObjectsParams struct {
	BucketName string `json:"name,omitempty" jsonschema:"description=Name of the bucket to delete objects from"`
}

type deleteObjectsError struct {
	uri string
	err error
}

type deleteObjectsErrors []deleteObjectsError

func (o deleteObjectsErrors) Error() string {
	var errorMsg string
	for _, objError := range o {
		errorMsg += fmt.Sprintf("%s - %s, ", objError.uri, objError.err)
	}
	// Remove trailing `, `
	if len(errorMsg) != 0 {
		errorMsg = errorMsg[:len(errorMsg)-2]
	}
	return fmt.Sprintf("failed to delete objects from bucket: %s", errorMsg)
}

func (o deleteObjectsErrors) HasError() bool {
	return len(o) != 0
}

func newDeleteRequest(ctx context.Context, cfg Config, pathURIs ...string) (*http.Request, error) {
	host := BuildHost(cfg)
	url, err := url.JoinPath(host, pathURIs...)
	if err != nil {
		return nil, core.UsageError{Err: err}
	}
	return http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
}

// Deleting an object does not yield result except there is an error. So this processor will *Skip*
// success results and *Output* errors
func createObjectDeletionProcessor(cfg Config, bucketName string) pipeline.Processor[pipeline.WalkDirEntry, deleteObjectsError] {
	return func(ctx context.Context, dirEntry pipeline.WalkDirEntry) (deleteObjectsError, pipeline.ProcessStatus) {
		if err := dirEntry.Err(); err != nil {
			return deleteObjectsError{err: err}, pipeline.ProcessAbort
		}

		obj, ok := dirEntry.DirEntry().(*BucketContent)
		if !ok {
			return deleteObjectsError{err: fmt.Errorf("expected object, got directory")}, pipeline.ProcessAbort
		}

		objURI := path.Join(bucketName, obj.Key)
		_, err := Delete(
			ctx,
			DeleteObjectParams{Destination: objURI},
			cfg,
		)

		if err != nil {
			return deleteObjectsError{uri: objURI, err: err}, pipeline.ProcessOutput
		} else {
			deleteLogger().Infow("Deleted objects", "uri", URIPrefix+objURI)
			return deleteObjectsError{}, pipeline.ProcessSkip
		}
	}
}

func DeleteAllObjects(ctx context.Context, params DeleteObjectParams, cfg Config) (deleteObjectsErrors, error) {
	listParams := ListObjectsParams{
		Destination: params.BucketName,
		Recursive:   true,
		PaginationParams: PaginationParams{
			MaxItems: math.MaxInt64,
		},
	}

	objs := ListGenerator(ctx, listParams, cfg)

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

	deleteObjectsErrorChan := pipeline.ParallelProcess(ctx, cfg.Workers, objs, createObjectDeletionProcessor(cfg, params.BucketName), nil)

	// This cannot error, there is no cancel call in processor
	objErr, _ := pipeline.SliceItemConsumer[deleteObjectsErrors](ctx, deleteObjectsErrorChan)

	return objErr
}

func Delete(ctx context.Context, params DeleteObjectParams, cfg Config) (result core.Value, err error) {
	bucketURI, _ := strings.CutPrefix(params.Destination, URIPrefix)
	req, err := newDeleteRequest(ctx, cfg, bucketURI)
	if err != nil {
		return nil, err
	}

	result, _, err = SendRequest[core.Value](ctx, req)
	return
}
