/*
Executor: list

# Summary

# List Rules

# Description

Returns a list of rules for a provided security_group_id

Version: 1.111.0

import "magalu.cloud/lib/products/network/rules"
*/
package rules

import (
	"context"

	mgcCore "magalu.cloud/core"
	mgcClient "magalu.cloud/lib"
	mgcHelpers "magalu.cloud/lib/helpers"
)

type ListParameters struct {
	SecurityGroupId string `json:"security_group_id"`
}

type ListConfigs struct {
	Env       string `json:"env,omitempty"`
	Region    string `json:"region,omitempty"`
	ServerUrl string `json:"serverUrl,omitempty"`
}

type ListResult struct {
	Rules ListResultRules `json:"rules"`
}

type ListResultRulesItem struct {
	CreatedAt       *string `json:"created_at,omitempty"`
	Direction       *string `json:"direction,omitempty"`
	Error           *string `json:"error,omitempty"`
	Ethertype       *string `json:"ethertype,omitempty"`
	Id              *string `json:"id,omitempty"`
	PortRangeMax    *int    `json:"port_range_max,omitempty"`
	PortRangeMin    *int    `json:"port_range_min,omitempty"`
	Protocol        *string `json:"protocol,omitempty"`
	RemoteGroupId   *string `json:"remote_group_id,omitempty"`
	RemoteIpPrefix  *string `json:"remote_ip_prefix,omitempty"`
	SecurityGroupId *string `json:"security_group_id,omitempty"`
	Status          string  `json:"status"`
}

type ListResultRules []ListResultRulesItem

func List(
	client *mgcClient.Client,
	ctx context.Context,
	parameters ListParameters,
	configs ListConfigs,
) (
	result ListResult,
	err error,
) {
	exec, ctx, err := mgcHelpers.PrepareExecutor("List", mgcCore.RefPath("/network/rules/list"), client, ctx)
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