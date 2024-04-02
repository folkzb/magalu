/*
Executor: create

# Summary

# Create Port

# Description

Create a Port with provided vpc_id and x-tenant-id. You can provide a list of security_groups_id or subnets

Version: 1.111.0

import "magalu.cloud/lib/products/network/ports"
*/
package ports

import (
	"context"

	mgcCore "magalu.cloud/core"
	mgcClient "magalu.cloud/lib"
	mgcHelpers "magalu.cloud/lib/helpers"
)

type CreateParameters struct {
	HasPip           *bool                             `json:"has_pip,omitempty"`
	HasSg            *bool                             `json:"has_sg,omitempty"`
	Name             string                            `json:"name"`
	SecurityGroupsId *CreateParametersSecurityGroupsId `json:"security_groups_id,omitempty"`
	Subnets          *CreateParametersSubnets          `json:"subnets,omitempty"`
	VpcId            string                            `json:"vpc_id"`
}

type CreateParametersSecurityGroupsId []string

type CreateParametersSubnets []string

type CreateConfigs struct {
	Env       string `json:"env,omitempty"`
	Region    string `json:"region,omitempty"`
	ServerUrl string `json:"serverUrl,omitempty"`
}

type CreateResult struct {
	Id string `json:"id"`
}

func Create(
	client *mgcClient.Client,
	ctx context.Context,
	parameters CreateParameters,
	configs CreateConfigs,
) (
	result CreateResult,
	err error,
) {
	exec, ctx, err := mgcHelpers.PrepareExecutor("Create", mgcCore.RefPath("/network/ports/create"), client, ctx)
	if err != nil {
		return
	}

	var p mgcCore.Parameters
	if p, err = mgcHelpers.ConvertParameters[CreateParameters](parameters); err != nil {
		return
	}

	var c mgcCore.Configs
	if c, err = mgcHelpers.ConvertConfigs[CreateConfigs](configs); err != nil {
		return
	}

	r, err := exec.Execute(ctx, p, c)
	if err != nil {
		return
	}
	return mgcHelpers.ConvertResult[CreateResult](r)
}

// TODO: links
// TODO: related
