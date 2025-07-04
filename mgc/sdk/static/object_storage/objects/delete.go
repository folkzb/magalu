package objects

import (
	"context"
	"fmt"
	"time"

	"github.com/MagaluCloud/magalu/mgc/core"
	"github.com/MagaluCloud/magalu/mgc/core/utils"
	"github.com/MagaluCloud/magalu/mgc/sdk/static/object_storage/common"
)

var getDelete = utils.NewLazyLoader[core.Executor](func() core.Executor {
	var exec core.Executor = core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "delete",
			Description: "Delete an object from a bucket",
		},
		deleteObject,
	)
	exec = core.NewExecuteFormat(exec, func(exec core.Executor, result core.Result) string {
		return fmt.Sprintf("Deleted object %q", result.Source().Parameters["dst"])
	})
	exec = core.NewPromptInputExecutor(
		exec,
		core.NewPromptInput(
			`This command will delete the object at {{.confirmationValue}}, and its result is NOT reversible.
Please confirm by retyping: {{.confirmationValue}}`,
			"{{.parameters.dst}}",
		),
	)

	return exec
})

func deleteObject(ctx context.Context, params common.DeleteObjectParams, cfg common.Config) (core.Value, error) {
	retries := cfg.Retries
	if retries <= 0 {
		retries = 0
	}
	backoff := 500 * time.Millisecond

	var err error
	for i := 0; i <= retries; i++ {
		err = common.Delete(ctx, params, cfg)
		if err == nil {
			return nil, nil
		}

		if isTemporaryErr(err) && i < retries {
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		return nil, err
	}

	return nil, err
}
