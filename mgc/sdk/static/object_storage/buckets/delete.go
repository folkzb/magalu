package buckets

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"go.uber.org/zap"
	"magalu.cloud/core"
	"magalu.cloud/core/pipeline"
	"magalu.cloud/sdk/static/object_storage/objects"
	"magalu.cloud/sdk/static/object_storage/s3"
)

var deleteBucketsLogger *zap.SugaredLogger

func deleteLogger() *zap.SugaredLogger {
	if deleteBucketsLogger == nil {
		deleteBucketsLogger = logger().Named("delete")
	}
	return deleteBucketsLogger
}

type deleteParams struct {
	Name string `json:"name" jsonschema:"description=Name of the bucket to be deleted"`
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

func newDelete() core.Executor {
	executor := core.NewStaticExecute(
		"delete",
		"",
		"Delete a bucket",
		delete,
	)

	msg := "This command will delete bucket {{.parameters.name}}, and it's result is NOT reversible."

	cExecutor := core.NewConfirmableExecutor(
		executor,
		core.ConfirmPromptWithTemplate(msg),
	)

	return core.NewExecuteResultOutputOptions(cExecutor, func(exec core.Executor, result core.Result) string {
		return "template=Deleted bucket {{.name}}\n"
	})
}

func newDeleteRequest(ctx context.Context, cfg s3.Config, pathURIs ...string) (*http.Request, error) {
	host := s3.BuildHost(cfg)
	url, err := url.JoinPath(host, pathURIs...)
	if err != nil {
		return nil, err
	}
	return http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
}

// Deleting an object does not yield result except there is an error. So this processor will *Skip*
// success results and *Output* errors
func createObjectDeletionProcessor(cfg s3.Config, bucketName string) pipeline.Processor[*objects.BucketContent, deleteObjectsError] {
	return func(ctx context.Context, obj *objects.BucketContent) (deleteObjectsError, pipeline.ProcessStatus) {
		objURI := path.Join(bucketName, obj.Key)
		_, err := objects.Delete(
			ctx,
			objects.DeleteObjectParams{Destination: objURI},
			cfg,
		)

		if err != nil {
			return deleteObjectsError{uri: objURI, err: err}, pipeline.ProcessOutput
		} else {
			deleteLogger().Infow("Deleted objects", "uri", s3.URIPrefix+objURI)
			return deleteObjectsError{}, pipeline.ProcessSkip
		}
	}
}

func delete(ctx context.Context, params deleteParams, cfg s3.Config) (core.Value, error) {
	objs, err := objects.List(ctx, objects.ListObjectsParams{Destination: params.Name}, cfg)
	if err != nil {
		return nil, err
	}

	objChan := pipeline.SliceItemGenerator(ctx, objs.Contents)

	deleteObjectsErrorChan := pipeline.ParallelProcess(ctx, cfg.Workers, objChan, createObjectDeletionProcessor(cfg, params.Name), nil)

	// This cannot error, there is no cancel call in processor
	objErr, _ := pipeline.SliceItemConsumer[deleteObjectsErrors](ctx, deleteObjectsErrorChan)

	if objErr.HasError() {
		return nil, objErr
	}

	req, err := newDeleteRequest(ctx, cfg, params.Name)
	if err != nil {
		return nil, err
	}

	_, _, err = s3.SendRequest[core.Value](ctx, req)
	if err != nil {
		return nil, err
	}

	return params, nil
}
