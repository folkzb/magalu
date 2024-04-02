/*
Executor: copy-all

# Description

# Copy all objects from a bucket to another bucket

import "magalu.cloud/lib/products/object_storage/objects"
*/
package objects

import (
	"context"

	mgcCore "magalu.cloud/core"
	mgcClient "magalu.cloud/lib"
	mgcHelpers "magalu.cloud/lib/helpers"
)

type CopyAllParameters struct {
	Dst    string                  `json:"dst"`
	Filter CopyAllParametersFilter `json:"filter,omitempty"`
	Src    string                  `json:"src"`
}

type CopyAllParametersFilterItem struct {
	Exclude string `json:"exclude,omitempty"`
	Include string `json:"include,omitempty"`
}

type CopyAllParametersFilter []CopyAllParametersFilterItem

type CopyAllConfigs struct {
	ChunkSize int    `json:"chunkSize,omitempty"`
	Region    string `json:"region,omitempty"`
	ServerUrl string `json:"serverUrl,omitempty"`
	Workers   int    `json:"workers,omitempty"`
}

type CopyAllResult struct {
	Dst    string              `json:"dst"`
	Filter CopyAllResultFilter `json:"filter,omitempty"`
	Src    string              `json:"src"`
}

type CopyAllResultFilterItem struct {
	Exclude string `json:"exclude,omitempty"`
	Include string `json:"include,omitempty"`
}

type CopyAllResultFilter []CopyAllResultFilterItem

func CopyAll(
	client *mgcClient.Client,
	ctx context.Context,
	parameters CopyAllParameters,
	configs CopyAllConfigs,
) (
	result CopyAllResult,
	err error,
) {
	exec, ctx, err := mgcHelpers.PrepareExecutor("CopyAll", mgcCore.RefPath("/object-storage/objects/copy-all"), client, ctx)
	if err != nil {
		return
	}

	var p mgcCore.Parameters
	if p, err = mgcHelpers.ConvertParameters[CopyAllParameters](parameters); err != nil {
		return
	}

	var c mgcCore.Configs
	if c, err = mgcHelpers.ConvertConfigs[CopyAllConfigs](configs); err != nil {
		return
	}

	r, err := exec.Execute(ctx, p, c)
	if err != nil {
		return
	}
	return mgcHelpers.ConvertResult[CopyAllResult](r)
}

// TODO: links
// TODO: related