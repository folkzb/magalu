/*
Executor: list

# Summary

Lists all snapshots in the current tenant.

# Description

List all snapshots in the current tenant which is logged in.

#### Notes
- You can use the **expand** argument to get more details from the inner objects
like image and machine types.

Version: 1.199.0

import "magalu.cloud/lib/products/virtual_machine/snapshots/v1_snapshots"
*/
package v1Snapshots

import (
	"context"

	mgcCore "magalu.cloud/core"
	mgcClient "magalu.cloud/lib"
	mgcHelpers "magalu.cloud/lib/helpers"
)

type ListParameters struct {
	Limit  int                  `json:"_limit,omitempty"`
	Offset int                  `json:"_offset,omitempty"`
	Sort   string               `json:"_sort,omitempty"`
	Expand ListParametersExpand `json:"expand,omitempty"`
}

type ListParametersExpand []string

type ListConfigs struct {
	Env       string `json:"env,omitempty"`
	Region    string `json:"region,omitempty"`
	ServerUrl string `json:"serverUrl,omitempty"`
}

type ListResult struct {
	Snapshots ListResultSnapshots `json:"snapshots"`
}

type ListResultSnapshotsItem struct {
	CreatedAt string                          `json:"created_at"`
	Id        string                          `json:"id"`
	Instance  ListResultSnapshotsItemInstance `json:"instance"`
	Name      *string                         `json:"name,omitempty"`
	Size      int                             `json:"size"`
	State     string                          `json:"state"`
	Status    string                          `json:"status"`
	UpdatedAt *string                         `json:"updated_at,omitempty"`
}

type ListResultSnapshotsItemInstance struct {
	Id          string                                     `json:"id"`
	Image       ListResultSnapshotsItemInstanceImage       `json:"image"`
	MachineType ListResultSnapshotsItemInstanceMachineType `json:"machine_type"`
}

// any of: ListResultSnapshotsItemInstanceImage0, ListResultSnapshotsItemInstanceImage1
type ListResultSnapshotsItemInstanceImage struct {
	ListResultSnapshotsItemInstanceImage0 `json:",squash"` // nolint
	ListResultSnapshotsItemInstanceImage1 `json:",squash"` // nolint
}

type ListResultSnapshotsItemInstanceImage0 struct {
	Id string `json:"id"`
}

type ListResultSnapshotsItemInstanceImage1 struct {
	Id       string  `json:"id"`
	Name     string  `json:"name"`
	Platform *string `json:"platform,omitempty"`
}

// any of: ListResultSnapshotsItemInstanceMachineType0, ListResultSnapshotsItemInstanceMachineType1
type ListResultSnapshotsItemInstanceMachineType struct {
	ListResultSnapshotsItemInstanceMachineType0 `json:",squash"` // nolint
	ListResultSnapshotsItemInstanceMachineType1 `json:",squash"` // nolint
}

type ListResultSnapshotsItemInstanceMachineType0 struct {
	Id string `json:"id"`
}

type ListResultSnapshotsItemInstanceMachineType1 struct {
	Disk  int    `json:"disk"`
	Id    string `json:"id"`
	Name  string `json:"name"`
	Ram   int    `json:"ram"`
	Vcpus int    `json:"vcpus"`
}

type ListResultSnapshots []ListResultSnapshotsItem

func List(
	client *mgcClient.Client,
	ctx context.Context,
	parameters ListParameters,
	configs ListConfigs,
) (
	result ListResult,
	err error,
) {
	exec, ctx, err := mgcHelpers.PrepareExecutor("List", mgcCore.RefPath("/virtual-machine/snapshots/v1-snapshots/list"), client, ctx)
	if err != nil {
		return
	}

	var p mgcCore.Parameters
	if p, err = mgcHelpers.ConvertParameters[ListParameters](parameters); err != nil {
		return
	}

	var c mgcCore.Configs
	if c, err = mgcHelpers.ConvertConfigs[ListConfigs](configs); err != nil {
		return
	}

	r, err := exec.Execute(ctx, p, c)
	if err != nil {
		return
	}
	return mgcHelpers.ConvertResult[ListResult](r)
}

// TODO: links
// TODO: related