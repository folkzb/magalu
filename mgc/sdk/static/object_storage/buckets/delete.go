package buckets

import (
	"context"
	"fmt"

	"github.com/MagaluCloud/magalu/mgc/core"
	"github.com/MagaluCloud/magalu/mgc/core/utils"
	"github.com/MagaluCloud/magalu/mgc/sdk/static/object_storage/common"
	"go.uber.org/zap"
)

var deleteBucketsLogger *zap.SugaredLogger

func deleteLogger() *zap.SugaredLogger {
	if deleteBucketsLogger == nil {
		deleteBucketsLogger = logger().Named("delete")
	}
	return deleteBucketsLogger
}

type deleteParams struct {
	BucketName common.BucketName `json:"bucket" jsonschema:"description=Name of the bucket to be deleted" mgc:"positional"`
	Recursive  bool              `json:"recursive" jsonschema:"description=This is a irreversible and destructive action. If set to true the bucket and its contents are deleted,default=false"`
}

var getDelete = utils.NewLazyLoader[core.Executor](func() core.Executor {
	var executor core.Executor = core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "delete",
			Description: "Delete an existing Bucket",
			// Scopes:      core.Scopes{"object-storage.write"},
		},
		deleteBucket,
	)

	executor = core.NewPromptInputExecutor(
		executor,
		core.NewPromptInput(
			`This command will delete bucket {{.confirmationValue}}, and its result is NOT reversible.
Please confirm by retyping: {{.confirmationValue}}`,
			"{{.parameters.bucket}}",
		),
	)

	return core.NewExecuteFormat(executor, func(exec core.Executor, result core.Result) string {
		return fmt.Sprintf("Deleted bucket %q", result.Source().Parameters["bucket"])
	})
})

func deleteBucket(ctx context.Context, params deleteParams, cfg common.Config) (core.Value, error) {
	logger := deleteLogger().Named("delete").With(
		"params", params,
		"cfg", cfg,
	)

	dst := params.BucketName.AsURI()
	err := common.DeleteBucket(ctx, common.DeleteBucketParams{Destination: dst, Recursive: params.Recursive}, cfg)
	if err != nil {
		return nil, err
	}

	logger.Info("Deleted bucket")
	return nil, err
}
