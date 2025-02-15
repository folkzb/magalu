/*
Executor: detach

# Summary

Detach the volume from an instance.

# Description

Detach a Volume from a Virtual Machine instance for the

	currently authenticated tenant.

The Volume detachment will be completed when the Volume state returns to

	"available," and the status becomes "completed".

#### Rules
- The Volume and the Virtual Machine must belong to the same tenant.
- Both the Volume and Virtual Machine must have the status "completed".
- The Volume's state must be "in-use".
- The Virtual Machine's state must be "stopped".

#### Notes
  - Verify the state and status of your Volume using the
    **block-storage volume get --id [uuid]** command.
  - Verify the state and status of your Virtual Machine using the
    **virtual-machine instances get --id [uuid]** command.
  - Ensure that any file systems on the device within your operating system are
    unmounted before detaching the Volume.

#### Troubleshooting
  - A failure during detachment can result in the Volume becoming stuck in the
    busy state. If this occurs, detachment may be delayed indefinitely until you
    unmount the Volume, force detachment, reboot the instance, or perform all
    three.

Version: v1

import "github.com/MagaluCloud/magalu/mgc/lib/products/block_storage/volumes"
*/
package volumes

import (
	"context"

	mgcCore "github.com/MagaluCloud/magalu/mgc/core"
	mgcHelpers "github.com/MagaluCloud/magalu/mgc/lib/helpers"
)

type DetachParameters struct {
	Id string `json:"id"`
}

type DetachConfigs struct {
	Env       *string `json:"env,omitempty"`
	Region    *string `json:"region,omitempty"`
	ServerUrl *string `json:"serverUrl,omitempty"`
}

func (s *service) Detach(
	parameters DetachParameters,
	configs DetachConfigs,
) (
	err error,
) {
	exec, ctx, err := mgcHelpers.PrepareExecutor("Detach", mgcCore.RefPath("/block-storage/volumes/detach"), s.client, s.ctx)
	if err != nil {
		return
	}

	var p mgcCore.Parameters
	if p, err = mgcHelpers.ConvertParameters[DetachParameters](parameters); err != nil {
		return
	}

	var c mgcCore.Configs
	if c, err = mgcHelpers.ConvertConfigs[DetachConfigs](configs); err != nil {
		return
	}

	_, err = exec.Execute(ctx, p, c)
	return
}

// Context from caller is used to allow cancellation of long-running requests
func (s *service) DetachContext(
	ctx context.Context,
	parameters DetachParameters,
	configs DetachConfigs,
) (
	err error,
) {
	exec, ctx, err := mgcHelpers.PrepareExecutor("Detach", mgcCore.RefPath("/block-storage/volumes/detach"), s.client, ctx)
	if err != nil {
		return
	}

	var p mgcCore.Parameters
	if p, err = mgcHelpers.ConvertParameters[DetachParameters](parameters); err != nil {
		return
	}

	var c mgcCore.Configs
	if c, err = mgcHelpers.ConvertConfigs[DetachConfigs](configs); err != nil {
		return
	}

	sdkConfig := s.client.Sdk().Config().TempConfig()
	if c["serverUrl"] == nil && sdkConfig["serverUrl"] != nil {
		c["serverUrl"] = sdkConfig["serverUrl"]
	}

	if c["env"] == nil && sdkConfig["env"] != nil {
		c["env"] = sdkConfig["env"]
	}

	if c["region"] == nil && sdkConfig["region"] != nil {
		c["region"] = sdkConfig["region"]
	}

	_, err = exec.Execute(ctx, p, c)
	return
}

// TODO: links
// TODO: related
