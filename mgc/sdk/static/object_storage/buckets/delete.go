package buckets

import (
	"context"

	"go.uber.org/zap"
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/common"
)

var deleteBucketsLogger *zap.SugaredLogger

func deleteLogger() *zap.SugaredLogger {
	if deleteBucketsLogger == nil {
		deleteBucketsLogger = logger().Named("delete")
	}
	return deleteBucketsLogger
}

type deleteParams struct {
	BucketName string `json:"bucket" jsonschema:"description=Name of the bucket to be deleted" mgc:"positional"`
}

var getDelete = utils.NewLazyLoader[core.Executor](func() core.Executor {
	executor := core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "delete",
			Description: "Delete an existing Bucket",
		},
		delete,
	)

	msg := "This command will delete bucket {{.parameters.name}}, and its result is NOT reversible."

	cExecutor := core.NewConfirmableExecutor(
		executor,
		core.ConfirmPromptWithTemplate(msg),
	)

	return core.NewExecuteResultOutputOptions(cExecutor, func(exec core.Executor, result core.Result) string {
		return "template=Deleted bucket {{.name}}\n"
	})
})

func delete(ctx context.Context, params deleteParams, cfg common.Config) (core.Value, error) {
	logger := deleteLogger().Named("delete").With(
		"params", params,
		"cfg", cfg,
	)

	objErr, err := common.DeleteAllObjects(ctx, common.DeleteAllObjectsParams{BucketName: params.BucketName, BatchSize: common.MaxBatchSize}, cfg)
	if err != nil {
		return nil, err
	}

	if objErr.HasError() {
		return nil, objErr
	}

	result, err := common.Delete(ctx, common.DeleteObjectParams{Destination: params.BucketName}, cfg)
	if err != nil {
		return nil, err
	}

	logger.Info("Deleted bucket")
	return result, nil
}
