/*
Executor: get

# Summary

# VPC Details

# Description

# Return a VPC details

Version: 1.111.0

import "magalu.cloud/lib/products/network/vpcs"
*/
package vpcs

import (
	"context"

	mgcCore "magalu.cloud/core"
	mgcClient "magalu.cloud/lib"
	mgcHelpers "magalu.cloud/lib/helpers"
)

type GetParameters struct {
	VpcId string `json:"vpc_id"`
}

type GetConfigs struct {
	Env       string `json:"env,omitempty"`
	Region    string `json:"region,omitempty"`
	ServerUrl string `json:"serverUrl,omitempty"`
}

type GetResult struct {
	CreatedAt       *string                 `json:"created_at,omitempty"`
	Description     *string                 `json:"description,omitempty"`
	ExternalNetwork *string                 `json:"external_network,omitempty"`
	Id              *string                 `json:"id,omitempty"`
	IsDefault       *bool                   `json:"is_default,omitempty"`
	Name            *string                 `json:"name,omitempty"`
	NetworkId       *string                 `json:"network_id,omitempty"`
	RouterId        *string                 `json:"router_id,omitempty"`
	SecurityGroups  GetResultSecurityGroups `json:"security_groups,omitempty"`
	Subnets         *GetResultSubnets       `json:"subnets,omitempty"`
	TenantId        *string                 `json:"tenant_id,omitempty"`
	Updated         *string                 `json:"updated,omitempty"`
}

type GetResultSecurityGroups []string

type GetResultSubnets []string

func Get(
	client *mgcClient.Client,
	ctx context.Context,
	parameters GetParameters,
	configs GetConfigs,
) (
	result GetResult,
	err error,
) {
	exec, ctx, err := mgcHelpers.PrepareExecutor("Get", mgcCore.RefPath("/network/vpcs/get"), client, ctx)
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