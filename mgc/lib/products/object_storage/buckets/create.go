/*
Executor: create

# Summary

# Create a new Bucket

# Description

Buckets are "containers" that are able to store various Objects inside

import "github.com/MagaluCloud/magalu/mgc/lib/products/object_storage/buckets"
*/
package buckets

import (
	"context"

	mgcCore "github.com/MagaluCloud/magalu/mgc/core"
	mgcHelpers "github.com/MagaluCloud/magalu/mgc/lib/helpers"
)

type CreateParameters struct {
	AuthenticatedRead *bool                             `json:"authenticated_read,omitempty"`
	AwsExecRead       *bool                             `json:"aws_exec_read,omitempty"`
	Bucket            string                            `json:"bucket"`
	BucketIsPrefix    bool                              `json:"bucket_is_prefix"`
	EnableVersioning  *bool                             `json:"enable_versioning,omitempty"`
	GrantFullControl  *CreateParametersGrantFullControl `json:"grant_full_control,omitempty"`
	GrantRead         *CreateParametersGrantRead        `json:"grant_read,omitempty"`
	GrantReadAcp      *CreateParametersGrantReadAcp     `json:"grant_read_acp,omitempty"`
	GrantWrite        *CreateParametersGrantWrite       `json:"grant_write,omitempty"`
	GrantWriteAcp     *CreateParametersGrantWriteAcp    `json:"grant_write_acp,omitempty"`
	Private           *bool                             `json:"private,omitempty"`
	PublicRead        *bool                             `json:"public_read,omitempty"`
	PublicReadWrite   *bool                             `json:"public_read_write,omitempty"`
}

type CreateParametersGrantFullControlItem struct {
	Id string `json:"id"`
}

type CreateParametersGrantFullControl []CreateParametersGrantFullControlItem

type CreateParametersGrantRead []CreateParametersGrantFullControlItem

type CreateParametersGrantReadAcp []CreateParametersGrantFullControlItem

type CreateParametersGrantWrite []CreateParametersGrantFullControlItem

type CreateParametersGrantWriteAcp []CreateParametersGrantFullControlItem

type CreateConfigs struct {
	ChunkSize *int    `json:"chunkSize,omitempty"`
	Region    *string `json:"region,omitempty"`
	ServerUrl *string `json:"serverUrl,omitempty"`
	Workers   *int    `json:"workers,omitempty"`
}

type CreateResult struct {
	AuthenticatedRead *bool                         `json:"authenticated_read,omitempty"`
	AwsExecRead       *bool                         `json:"aws_exec_read,omitempty"`
	Bucket            string                        `json:"bucket"`
	BucketIsPrefix    bool                          `json:"bucket_is_prefix"`
	EnableVersioning  *bool                         `json:"enable_versioning,omitempty"`
	GrantFullControl  *CreateResultGrantFullControl `json:"grant_full_control,omitempty"`
	GrantRead         *CreateResultGrantRead        `json:"grant_read,omitempty"`
	GrantReadAcp      *CreateResultGrantReadAcp     `json:"grant_read_acp,omitempty"`
	GrantWrite        *CreateResultGrantWrite       `json:"grant_write,omitempty"`
	GrantWriteAcp     *CreateResultGrantWriteAcp    `json:"grant_write_acp,omitempty"`
	Private           *bool                         `json:"private,omitempty"`
	PublicRead        *bool                         `json:"public_read,omitempty"`
	PublicReadWrite   *bool                         `json:"public_read_write,omitempty"`
}

type CreateResultGrantFullControlItem struct {
	Id string `json:"id"`
}

type CreateResultGrantFullControl []CreateResultGrantFullControlItem

type CreateResultGrantRead []CreateResultGrantFullControlItem

type CreateResultGrantReadAcp []CreateResultGrantFullControlItem

type CreateResultGrantWrite []CreateResultGrantFullControlItem

type CreateResultGrantWriteAcp []CreateResultGrantFullControlItem

func (s *service) Create(
	parameters CreateParameters,
	configs CreateConfigs,
) (
	result CreateResult,
	err error,
) {
	exec, ctx, err := mgcHelpers.PrepareExecutor("Create", mgcCore.RefPath("/object-storage/buckets/create"), s.client, s.ctx)
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

// Context from caller is used to allow cancellation of long-running requests
func (s *service) CreateContext(
	ctx context.Context,
	parameters CreateParameters,
	configs CreateConfigs,
) (
	result CreateResult,
	err error,
) {
	exec, ctx, err := mgcHelpers.PrepareExecutor("Create", mgcCore.RefPath("/object-storage/buckets/create"), s.client, ctx)
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

	r, err := exec.Execute(ctx, p, c)
	if err != nil {
		return
	}
	return mgcHelpers.ConvertResult[CreateResult](r)
}

// TODO: links
// TODO: related
