package objects

import (
	"context"

	"magalu.cloud/core"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/common"
)

var getDeleteAll = utils.NewLazyLoader[core.Executor](func() core.Executor {
	exec := core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "delete-all",
			Description: "Delete all objects from a bucket",
		},
		deleteAll,
	)

	msg := "This command will delete all objects at {{.parameters.name}}, and its result is NOT reversible."

	return core.NewConfirmableExecutor(
		exec,
		core.ConfirmPromptWithTemplate(msg),
	)
})

func deleteAll(ctx context.Context, params common.DeleteAllObjectsParams, cfg common.Config) (result core.Value, err error) {
	err = common.DeleteAllObjects(ctx, params, cfg)
	return
}
