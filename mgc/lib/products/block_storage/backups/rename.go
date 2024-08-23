/*
Executor: rename

# Summary

Rename a backup.

# Description

Patches a Backup for the currently authenticated tenant.

#### Rules
- The Backup name must be unique; otherwise, renaming will not be allowed.
- The Backup's state must be available.

#### Notes
  - Utilize the **block-storage backups list** command to retrieve a list of
    all Backups and obtain the ID of the Backup you wish to rename.

Version: v1

import "magalu.cloud/lib/products/block_storage/backups"
*/
package backups

import (
	mgcCore "magalu.cloud/core"
	mgcHelpers "magalu.cloud/lib/helpers"
)

type RenameParameters struct {
	Description *string `json:"description,omitempty"`
	Id          string  `json:"id"`
	Name        *string `json:"name,omitempty"`
}

type RenameConfigs struct {
	Env       *string `json:"env,omitempty"`
	Region    *string `json:"region,omitempty"`
	ServerUrl *string `json:"serverUrl,omitempty"`
}

func (s *service) Rename(
	parameters RenameParameters,
	configs RenameConfigs,
) (
	err error,
) {
	exec, ctx, err := mgcHelpers.PrepareExecutor("Rename", mgcCore.RefPath("/block-storage/backups/rename"), s.client, s.ctx)
	if err != nil {
		return
	}

	var p mgcCore.Parameters
	if p, err = mgcHelpers.ConvertParameters[RenameParameters](parameters); err != nil {
		return
	}

	var c mgcCore.Configs
	if c, err = mgcHelpers.ConvertConfigs[RenameConfigs](configs); err != nil {
		return
	}

	_, err = exec.Execute(ctx, p, c)
	return
}

// TODO: links
// TODO: related