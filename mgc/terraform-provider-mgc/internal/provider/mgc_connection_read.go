package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
)

type MgcConnectionRead struct {
	resourceName   tfName
	attrTree       resAttrInfoTree
	readConnection core.Executor
}

func newMgcConnectionRead(
	resourceName tfName,
	attrTree resAttrInfoTree,
	readConnection core.Executor,
) MgcOperation {
	return &MgcConnectionRead{
		resourceName:   resourceName,
		attrTree:       attrTree,
		readConnection: readConnection,
	}
}

func (o *MgcConnectionRead) WrapConext(ctx context.Context) context.Context {
	ctx = tflog.SetField(ctx, rpcField, "read")
	ctx = tflog.SetField(ctx, connectionResourceNameField, o.resourceName)
	return ctx
}

func (o *MgcConnectionRead) CollectParameters(ctx context.Context, _, plan TerraformParams) (core.Parameters, Diagnostics) {
	return loadMgcParamsFromState(ctx, o.readConnection.ParametersSchema(), o.attrTree, plan)
}

func (o *MgcConnectionRead) CollectConfigs(ctx context.Context, _, _ TerraformParams) (core.Configs, Diagnostics) {
	return getConfigs(ctx, o.readConnection.ConfigsSchema()), nil
}

func (o *MgcConnectionRead) ShouldRun(context.Context, core.Parameters, core.Configs) (run bool, d Diagnostics) {
	return true, d
}

func (o *MgcConnectionRead) Run(ctx context.Context, params core.Parameters, configs core.Configs) (core.ResultWithValue, Diagnostics) {
	return execute(ctx, o.resourceName, o.readConnection, params, configs)
}

func (o *MgcConnectionRead) PostRun(ctx context.Context, readResult core.ResultWithValue, state, plan TerraformParams, targetState *tfsdk.State) (core.ResultWithValue, bool, Diagnostics) {
	tflog.Info(ctx, "connection read")
	readResult, _, d := applyStateAfter(ctx, o.resourceName, o.attrTree, readResult, o.readConnection, targetState)
	return readResult, true, d
}

func (o *MgcConnectionRead) ReadResultSchema() *mgcSchemaPkg.Schema {
	return o.readConnection.ResultSchema()
}

func (o *MgcConnectionRead) ChainOperations(ctx context.Context, _ core.ResultWithValue, readResult ReadResult, state, plan TerraformParams) ([]MgcOperation, bool, Diagnostics) {
	return nil, false, nil
}

var _ MgcOperation = (*MgcConnectionRead)(nil)
var _ MgcReadOperation = (*MgcConnectionRead)(nil)
