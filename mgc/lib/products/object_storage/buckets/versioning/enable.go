/*
Executor: enable

# Description

# Enable versioning for a Bucket

import "magalu.cloud/lib/products/object_storage/buckets/versioning"
*/
package versioning

import (
	"context"

	mgcCore "magalu.cloud/core"
	mgcClient "magalu.cloud/lib"
	mgcHelpers "magalu.cloud/lib/helpers"
)

type EnableParameters struct {
	Bucket string `json:"bucket"`
}

type EnableConfigs struct {
	ChunkSize int    `json:"chunkSize,omitempty"`
	Region    string `json:"region,omitempty"`
	ServerUrl string `json:"serverUrl,omitempty"`
	Workers   int    `json:"workers,omitempty"`
}

type EnableResult any

func Enable(
	client *mgcClient.Client,
	ctx context.Context,
	parameters EnableParameters,
	configs EnableConfigs,
) (
	result EnableResult,
	err error,
) {
	exec, ctx, err := mgcHelpers.PrepareExecutor("Enable", mgcCore.RefPath("/object-storage/buckets/versioning/enable"), client, ctx)
	if err != nil {
		return
	}

	var p mgcCore.Parameters
	if p, err = mgcHelpers.ConvertParameters[EnableParameters](parameters); err != nil {
		return
	}

	var c mgcCore.Configs
	if c, err = mgcHelpers.ConvertConfigs[EnableConfigs](configs); err != nil {
		return
	}

	r, err := exec.Execute(ctx, p, c)
	if err != nil {
		return
	}
	return mgcHelpers.ConvertResult[EnableResult](r)
}

// TODO: links
// TODO: related