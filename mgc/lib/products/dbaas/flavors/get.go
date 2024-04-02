/*
Executor: get

# Summary

Flavor detail.

# Description

Returns a flavor detail.

Version: 1.17.2

import "magalu.cloud/lib/products/dbaas/flavors"
*/
package flavors

import (
	"context"

	mgcCore "magalu.cloud/core"
	mgcClient "magalu.cloud/lib"
	mgcHelpers "magalu.cloud/lib/helpers"
)

type GetParameters struct {
	FlavorId string `json:"flavor_id"`
}

type GetConfigs struct {
	Env       string `json:"env,omitempty"`
	Region    string `json:"region,omitempty"`
	ServerUrl string `json:"serverUrl,omitempty"`
}

type GetResult struct {
	FamilyDescription string `json:"family_description"`
	FamilySlug        string `json:"family_slug"`
	Id                string `json:"id"`
	Label             string `json:"label"`
	Name              string `json:"name"`
	Ram               string `json:"ram"`
	Size              string `json:"size"`
	SkuReplica        string `json:"sku_replica"`
	SkuSource         string `json:"sku_source"`
	Vcpu              string `json:"vcpu"`
}

func Get(
	client *mgcClient.Client,
	ctx context.Context,
	parameters GetParameters,
	configs GetConfigs,
) (
	result GetResult,
	err error,
) {
	exec, ctx, err := mgcHelpers.PrepareExecutor("Get", mgcCore.RefPath("/dbaas/flavors/get"), client, ctx)
	if err != nil {
		return
	}

	var p mgcCore.Parameters
	if p, err = mgcHelpers.ConvertParameters[GetParameters](parameters); err != nil {
		return
	}

	var c mgcCore.Configs
	if c, err = mgcHelpers.ConvertConfigs[GetConfigs](configs); err != nil {
		return
	}

	r, err := exec.Execute(ctx, p, c)
	if err != nil {
		return
	}
	return mgcHelpers.ConvertResult[GetResult](r)
}

// TODO: links
// TODO: related