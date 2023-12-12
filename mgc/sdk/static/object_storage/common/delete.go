package common

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"math"
	"net/http"
	"net/url"

	"go.uber.org/zap"
	"magalu.cloud/core"
	"magalu.cloud/core/pipeline"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
)

var deleteObjectsLogger *zap.SugaredLogger

func deleteLogger() *zap.SugaredLogger {
	if deleteObjectsLogger == nil {
		deleteObjectsLogger = logger().Named("delete")
	}
	return deleteObjectsLogger
}

type DeleteObjectParams struct {
	Destination mgcSchemaPkg.URI `json:"dst,omitempty" jsonschema:"description=Path of the object to be deleted,example=s3://bucket1/file1" mgc:"positional"`
}

type DeleteAllObjectsParams struct {
	BucketName   BucketName       `json:"bucket,omitempty" jsonschema:"description=Name of the bucket to delete objects from" mgc:"positional"`
	BatchSize    int              `json:"batch_size,omitempty" jsonschema:"description=Limit of items per batch to delete,default=1000,minimum=1,maximum=1000" example:"1000"`
	FilterParams `json:",squash"` // nolint
}

type objectIdentifier struct {
	Key string `xml:"Key"`
}

type deleteBatchRequestBody struct {
	XMLName struct{}           `xml:"Delete"`
	Objects []objectIdentifier `xml:"Object"`
}

func newDeleteRequest(ctx context.Context, cfg Config, pathURIs ...string) (*http.Request, error) {
	host := BuildHost(cfg)
	url, err := url.JoinPath(host, pathURIs...)
	if err != nil {
		return nil, core.UsageError{Err: err}
	}
	return http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
}

func newDeleteBatchRequest(ctx context.Context, cfg Config, bucketName string, objKeys []objectIdentifier) (*http.Request, error) {
	host := BuildHost(cfg)
	url, err := url.JoinPath(host, bucketName)
	if err != nil {
		return nil, core.UsageError{Err: err}
	}
	body := deleteBatchRequestBody{
		Objects: objKeys,
	}
	marshalledBody, err := xml.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(marshalledBody))
	if err != nil {
		return nil, err
	}

	query := req.URL.Query()
	query.Set("delete", "")
	req.URL.RawQuery = query.Encode()

	return req, nil
}

// Deleting an object does not yield result except there is an error. So this processor will *Skip*
// success results and *Output* errors
func createObjectDeletionProcessor(cfg Config, bucketName BucketName) pipeline.Processor[[]pipeline.WalkDirEntry, error] {
	return func(ctx context.Context, dirEntries []pipeline.WalkDirEntry) (error, pipeline.ProcessStatus) {
		var objIdentifiers []objectIdentifier

		for _, dirEntry := range dirEntries {
			if err := dirEntry.Err(); err != nil {
				return &ObjectError{Err: err}, pipeline.ProcessAbort
			}

			obj, ok := dirEntry.DirEntry().(*BucketContent)
			if !ok {
				return &ObjectError{Err: fmt.Errorf("expected object, got directory")}, pipeline.ProcessAbort
			}

			objIdentifiers = append(objIdentifiers, objectIdentifier{Key: obj.Key})
		}

		req, err := newDeleteBatchRequest(ctx, cfg, bucketName.String(), objIdentifiers)
		if err != nil {
			return &ObjectError{Err: err}, pipeline.ProcessAbort
		}

		_, _, err = SendRequest[any](ctx, req)

		if err != nil {
			return &ObjectError{Url: mgcSchemaPkg.URI(bucketName), Err: err}, pipeline.ProcessOutput
		} else {
			deleteLogger().Infow("Deleted objects", "uri", URIPrefix+bucketName)
			return nil, pipeline.ProcessOutput
		}
	}
}

func DeleteAllObjects(ctx context.Context, params DeleteAllObjectsParams, cfg Config) error {
	listParams := ListObjectsParams{
		Destination: params.BucketName.AsURI(),
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

	if params.BatchSize < minBatchSize || params.BatchSize > MaxBatchSize {
		return core.UsageError{Err: fmt.Errorf("invalid item limit per request BatchSize, must not be lower than %d and must not be higher than %d: %d", minBatchSize, MaxBatchSize, params.BatchSize)}
	}

	objsBatch := pipeline.Batch(ctx, objs, params.BatchSize)
	deleteObjectsErrorChan := pipeline.ParallelProcess(ctx, cfg.Workers, objsBatch, createObjectDeletionProcessor(cfg, params.BucketName), nil)
	nonNilErrorsChan := pipeline.Filter(ctx, deleteObjectsErrorChan, pipeline.FilterNonNil[error]{})

	// This cannot error, there is no cancel call in processor
	objErr, _ := pipeline.SliceItemConsumer[utils.MultiError](ctx, nonNilErrorsChan)
	if len(objErr) > 0 {
		return objErr
	}

	return nil
}

func Delete(ctx context.Context, params DeleteObjectParams, cfg Config) (err error) {
	bucketPath := params.Destination.Path()
	req, err := newDeleteRequest(ctx, cfg, bucketPath)
	if err != nil {
		return
	}

	_, _, err = SendRequest[core.Value](ctx, req)
	return
}
