package common

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"math"
	"net/http"

	"go.uber.org/zap"
	"magalu.cloud/core"
	"magalu.cloud/core/pipeline"
	"magalu.cloud/core/progress_report"
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
	Destination mgcSchemaPkg.URI `json:"dst" jsonschema:"description=Path of the object to be deleted,example=s3://bucket1/file.txt" mgc:"positional"`
}

type DeleteAllObjectsInBucketParams struct {
	BucketName BucketName       `json:"bucket" jsonschema:"description=Name of the bucket to delete objects from" mgc:"positional"`
	BatchSize  int              `json:"batch_size,omitempty" jsonschema:"description=Limit of items per batch to delete,default=1000,minimum=1,maximum=1000" example:"1000"`
	Filters    `json:",squash"` // nolint
}

type objectIdentifier struct {
	Key string `xml:"Key"`
}

type deleteBatchRequestBody struct {
	XMLName struct{}           `xml:"Delete"`
	Objects []objectIdentifier `xml:"Object"`
}

func newDeleteRequest(ctx context.Context, cfg Config, dst mgcSchemaPkg.URI) (*http.Request, error) {
	host, err := BuildBucketHostWithPath(cfg, NewBucketNameFromURI(dst), dst.Path())
	if err != nil {
		return nil, core.UsageError{Err: err}
	}
	return http.NewRequestWithContext(ctx, http.MethodDelete, string(host), nil)
}

func newDeleteBatchRequest(ctx context.Context, cfg Config, bucketName BucketName, objKeys []objectIdentifier) (*http.Request, error) {
	host, err := BuildBucketHost(cfg, bucketName)
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, string(host), bytes.NewBuffer(marshalledBody))
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
func CreateObjectDeletionProcessor(cfg Config, bucketName BucketName, reportChan chan<- DeleteProgressReport) pipeline.Processor[[]pipeline.WalkDirEntry, error] {
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

		req, err := newDeleteBatchRequest(ctx, cfg, bucketName, objIdentifiers)
		if err != nil {
			return &ObjectError{Err: err}, pipeline.ProcessAbort
		}

		resp, err := SendRequest(ctx, req)

		reportChan <- DeleteProgressReport{uint64(len(dirEntries)), 0, err}

		if err != nil {
			return &ObjectError{Url: mgcSchemaPkg.URI(bucketName), Err: err}, pipeline.ProcessOutput
		}

		err = ExtractErr(resp, req)
		if err != nil {
			return &ObjectError{Err: err}, pipeline.ProcessAbort
		}

		deleteLogger().Infow("Deleted objects", "uri", URIPrefix+bucketName)
		return nil, pipeline.ProcessOutput
	}
}

type DeleteProgressReport struct {
	files uint64
	total uint64
	err   error
}

func DeleteAllObjectsInBucket(ctx context.Context, params DeleteAllObjectsInBucketParams, cfg Config) error {
	dst := params.BucketName.AsURI()
	listParams := ListObjectsParams{
		Destination: dst,
		Recursive:   true,
		PaginationParams: PaginationParams{
			MaxItems: math.MaxInt64,
		},
	}

	reportProgress := progress_report.FromContext(ctx)
	reportChan := make(chan DeleteProgressReport)
	defer close(reportChan)

	onNewPage := func(objCount uint64) {
		reportChan <- DeleteProgressReport{0, objCount, nil}
	}

	objs := ListGenerator(ctx, listParams, cfg, onNewPage)
	objs = ApplyFilters(ctx, objs, params.FilterParams, nil)

	if params.BatchSize < MinBatchSize || params.BatchSize > MaxBatchSize {
		return core.UsageError{Err: fmt.Errorf("invalid item limit per request BatchSize, must not be lower than %d and must not be higher than %d: %d", MinBatchSize, MaxBatchSize, params.BatchSize)}
	}

	go ReportDeleteProgress(reportProgress, reportChan, params)

	objsBatch := pipeline.Batch(ctx, objs, params.BatchSize)
	deleteObjectsErrorChan := pipeline.ParallelProcess(ctx, cfg.Workers, objsBatch, CreateObjectDeletionProcessor(cfg, params.BucketName, reportChan), nil)
	deleteObjectsErrorChan = pipeline.Filter(ctx, deleteObjectsErrorChan, pipeline.FilterNonNil[error]{})

	// This cannot error, there is no cancel call in processor
	objErr, _ := pipeline.SliceItemConsumer[utils.MultiError](ctx, deleteObjectsErrorChan)
	if len(objErr) > 0 {
		return objErr
	}

	return nil
}

func ReportDeleteProgress(reportProgress progress_report.ReportProgress, reportChan <-chan DeleteProgressReport, p DeleteAllObjectsInBucketParams) {
	name := "Delete from " + p.BucketName.String()
	total := uint64(0)
	progress := uint64(0)

	// total here must be reported as one, otherwise the progress-bar shows
	// an animation we do not wish the user to see
	reportProgress(name, progress, 1, progress_report.UnitsNone, nil)

	var errors utils.MultiError
	for report := range reportChan {
		progress += report.files
		total += report.total

		if report.err != nil {
			errors = append(errors, report.err)
		}

		reportProgress(name, progress, total, progress_report.UnitsNone, nil)
	}

	if len(errors) > 0 {
		reportProgress(name, progress, total, progress_report.UnitsNone, errors)
		return
	}

	reportProgress(name, total, total, progress_report.UnitsNone, progress_report.ErrorProgressDone)
}

func Delete(ctx context.Context, params DeleteObjectParams, cfg Config) (err error) {
	req, err := newDeleteRequest(ctx, cfg, params.Destination)
	if err != nil {
		return
	}

	reportProgress := progress_report.FromContext(ctx)
	reportMsg := "Deleting object from bucket: " + params.Destination.String()
	progress := uint64(0)
	total := uint64(1)

	reportProgress(reportMsg, progress, progress, progress_report.UnitsNone, nil)

	resp, err := SendRequest(ctx, req)
	if err != nil {
		reportProgress(reportMsg, progress, progress, progress_report.UnitsNone, err)
		return
	}

	_, err = UnwrapResponse[core.Value](resp, req)
	if err != nil {
		reportProgress(reportMsg, progress, total, progress_report.UnitsNone, err)
		return
	}

	reportProgress(reportMsg, total, total, progress_report.UnitsNone, progress_report.ErrorProgressDone)

	return nil
}
