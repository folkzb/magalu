package objects

import (
	"context"

	"magalu.cloud/core"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/common"
)

var getDelete = utils.NewLazyLoader[core.Executor](func() core.Executor {
	exec := core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "delete",
			Description: "Delete an object from a bucket",
		},
		deleteObject,
	)

	msg := "This command will delete the object at {{.parameters.dst}}, and its result is NOT reversible."

	return core.NewConfirmableExecutor(
		exec,
		core.ConfirmPromptWithTemplate(msg),
	)
})

func deleteObject(ctx context.Context, params common.DeleteObjectParams, cfg common.Config) (result core.Value, err error) {
	err = common.Delete(ctx, params, cfg)
	return
}
