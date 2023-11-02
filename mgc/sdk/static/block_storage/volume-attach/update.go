package attachment

import (
	"context"

	"magalu.cloud/core"
	"magalu.cloud/core/utils"
)

type UpdateAttachVolumeParams struct {
	VolumeID         string `json:"id" jsonschema:"description=Block storage volume ID to be attached"`
	VirtualMachineID string `json:"virtual_machine_id" jsonschema:"description=ID of the virtual machine instance to attach the volume"`
}

var getUpdate = utils.NewLazyLoader[core.Executor](newUpdate)

func newUpdate() core.Executor {
	exec := core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "update",
			Description: "Update a block storage volume attachment",
		},
		update,
	)
	return core.NewExecuteResultOutputOptions(exec, func(exec core.Executor, result core.Result) string {
		return "No-op"
	})
}

func update(ctx context.Context, params UpdateAttachVolumeParams, cfg core.Configs) (core.Result, error) {
	// No update available
	return nil, nil
}
