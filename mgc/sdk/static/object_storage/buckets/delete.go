package buckets

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"path"

	"go.uber.org/zap"
	"magalu.cloud/core"
	"magalu.cloud/core/pipeline"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/common"
	"magalu.cloud/sdk/static/object_storage/objects"
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

var getDelete = utils.NewLazyLoader[core.Executor](newDelete)

func newDelete() core.Executor {
	executor := core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "delete",
			Description: "Delete an existing Bucket",
		},
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

func newDeleteRequest(ctx context.Context, cfg common.Config, pathURIs ...string) (*http.Request, error) {
	host := common.BuildHost(cfg)
	url, err := url.JoinPath(host, pathURIs...)
	if err != nil {
		return nil, core.UsageError{Err: err}
	}
	return http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
}

// Deleting an object does not yield result except there is an error. So this processor will *Skip*
// success results and *Output* errors
func createObjectDeletionProcessor(cfg common.Config, bucketName string) pipeline.Processor[*common.BucketContent, deleteObjectsError] {
	return func(ctx context.Context, obj *common.BucketContent) (deleteObjectsError, pipeline.ProcessStatus) {
		objURI := path.Join(bucketName, obj.Key)
		_, err := objects.Delete(
			ctx,
			objects.DeleteObjectParams{Destination: objURI},
			cfg,
		)

		if err != nil {
			return deleteObjectsError{uri: objURI, err: err}, pipeline.ProcessOutput
		} else {
			deleteLogger().Infow("Deleted objects", "uri", common.URIPrefix+objURI)
			return deleteObjectsError{}, pipeline.ProcessSkip
		}
	}
}

func delete(ctx context.Context, params deleteParams, cfg common.Config) (core.Value, error) {
	listParams := common.ListObjectsParams{
		Destination: params.Name,
		Recursive:   true,
		PaginationParams: common.PaginationParams{
			MaxItems: math.MaxInt64,
		},
	}
	objs, err := objects.List(ctx, listParams, cfg)
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

	_, _, err = common.SendRequest[core.Value](ctx, req)
	if err != nil {
		return nil, err
	}

	return params, nil
}
