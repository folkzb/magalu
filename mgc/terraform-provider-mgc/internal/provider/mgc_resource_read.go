package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core"
)

type MgcResourceRead struct {
	resourceName tfName
	attrTree     resAttrInfoTree
	operation    core.Executor
}

func newMgcResourceRead(resourceName tfName, attrTree resAttrInfoTree, operation core.Executor) *MgcResourceRead {
	return &MgcResourceRead{resourceName: resourceName, attrTree: attrTree, operation: operation}
}

func (o *MgcResourceRead) WrapConext(ctx context.Context) context.Context {
	ctx = tflog.SetField(ctx, rpcField, "read")
	ctx = tflog.SetField(ctx, resourceNameField, o.resourceName)
	return ctx
}

func (o *MgcResourceRead) CollectParameters(ctx context.Context, state, _ TerraformParams) (core.Parameters, Diagnostics) {
	return loadMgcParamsFromState(ctx, o.operation.ParametersSchema(), o.attrTree, state)
}

func (o *MgcResourceRead) CollectConfigs(ctx context.Context, _, _ TerraformParams) (core.Configs, Diagnostics) {
	return getConfigs(ctx, o.operation.ConfigsSchema()), nil
}

func (o *MgcResourceRead) ShouldRun(context.Context, core.Parameters, core.Configs) (run bool, d Diagnostics) {
	return true, d
}

func (o *MgcResourceRead) Run(ctx context.Context, params core.Parameters, configs core.Configs) (core.ResultWithValue, Diagnostics) {
	return execute(ctx, o.resourceName, o.operation, params, configs)
}

func (o *MgcResourceRead) PostRun(ctx context.Context, result core.ResultWithValue, state, plan TerraformParams, targetState *tfsdk.State) (core.ResultWithValue, bool, Diagnostics) {
	tflog.Info(ctx, "resource read")
	result, _, d := applyStateAfter(ctx, o.resourceName, o.attrTree, result, o.operation, targetState)
	return result, true, d
}

func (o *MgcResourceRead) ReadResultSchema() *mgcSchemaPkg.Schema {
	return o.operation.ResultSchema()
}

func (o *MgcResourceRead) ChainOperations(context.Context, core.ResultWithValue, ReadResult, TerraformParams, TerraformParams) ([]MgcOperation, bool, Diagnostics) {
	return nil, false, nil
}

var _ MgcOperation = (*MgcResourceRead)(nil)
