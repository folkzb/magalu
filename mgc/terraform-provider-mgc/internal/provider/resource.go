package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"magalu.cloud/core"
	mgcSdk "magalu.cloud/sdk"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &MgcResource{}
var _ resource.ResourceWithImportState = &MgcResource{}

// MgcResource defines the resource implementation.
type MgcResource struct {
	sdk        *mgcSdk.Sdk
	name       string
	group      mgcSdk.Grouper // TODO: is this needed?
	create     mgcSdk.Executor
	read       mgcSdk.Executor
	update     mgcSdk.Executor
	delete     mgcSdk.Executor
	inputAttr  mgcAttributes
	outputAttr mgcAttributes
	tfschema   *schema.Schema
}

// TODO: remove once we translate directly from mgcSdk.Schema
type VirtualMachineResourceModel struct {
	Id              types.String `tfsdk:"id"`
	InstanceID      types.String `tfsdk:"instance_id"`
	Name            types.String `tfsdk:"name"`
	Type            types.String `tfsdk:"type"`
	DesiredImage    types.String `tfsdk:"desired_image"`
	SSHKey          types.String `tfsdk:"key_name"`
	AllocFip        types.Bool   `tfsdk:"allocate_fip"`
	VCPUs           types.Int64  `tfsdk:"vcpus"`
	Memory          types.Int64  `tfsdk:"memory"`
	RootStorage     types.Int64  `tfsdk:"root_storage"`
	UserData        types.String `tfsdk:"user_data"`
	Zone            types.String `tfsdk:"availability_zone"`
	CurrentStatus   types.String `tfsdk:"current_status"`
	DesiredStatus   types.String `tfsdk:"desired_status"`
	PowerState      types.Int64  `tfsdk:"power_state"`
	PowerStateLabel types.String `tfsdk:"power_state_label"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
	Error           types.String `tfsdk:"error"`
	// Net      types.List   `tfsdk:"network_interfaces"`
}

func (r *MgcResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.name
}

func (r *MgcResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// TODO: Handle nullable values
	tflog.Debug(ctx, fmt.Sprintf("[resource] generating schema for `%s`", r.name))

	if r.tfschema == nil {
		tfs, d := r.generateTFSchema(ctx)
		resp.Diagnostics.Append(d...)
		if d.HasError() {
			return
		}

		tfs.MarkdownDescription = r.name
		r.tfschema = &tfs
	}

	resp.Schema = *r.tfschema
}

func (r *MgcResource) readMgcMap(mgcSchema *mgcSdk.Schema, ctx context.Context, tfState tfsdk.State, diag *diag.Diagnostics) map[string]any {
	conv := newTFStateConverter(ctx, diag, r.tfschema)
	return conv.readMgcMap(mgcSchema, r.inputAttr, tfState)
}

func (r *MgcResource) applyMgcInputMap(mgcMap map[string]any, ctx context.Context, tfState *tfsdk.State, diag *diag.Diagnostics) {
	conv := newTFStateConverter(ctx, diag, r.tfschema)
	conv.applyMgcMap(mgcMap, r.inputAttr, ctx, tfState, path.Empty())
}

func (r *MgcResource) applyMgcOutputMap(mgcMap map[string]any, ctx context.Context, tfState *tfsdk.State, diag *diag.Diagnostics) {
	conv := newTFStateConverter(ctx, diag, r.tfschema)
	conv.applyMgcMap(mgcMap, r.outputAttr, ctx, tfState, path.Empty())
}

func castToMap(result core.ResultWithValue, diag *diag.Diagnostics) (resultMap map[string]any, ok bool) {
	resultMap, ok = result.Value().(map[string]any)
	if !ok {
		diag.AddError(
			"Operation output mismatch",
			fmt.Sprintf("Unable to convert %v to map.", result),
		)
	}
	return
}

func (r *MgcResource) readResource(ctx context.Context, mgcState map[string]any, configs map[string]any, diag *diag.Diagnostics) map[string]any {
	exec := r.read

	params := map[string]any{}
	for k := range exec.ParametersSchema().Properties {
		if value, ok := mgcState[k]; ok {
			params[k] = value
		}
	}

	var result core.Result
	var err error

	tflog.Debug(ctx, fmt.Sprintf("[resource] reading new %s resource - request info with params: %+v and configs: %+v", r.name, params, configs))
	if tExec, ok := core.ExecutorAs[core.TerminatorExecutor](exec); ok {
		tflog.Debug(ctx, "[resource] running as TerminatorExecutor")
		result, err = tExec.ExecuteUntilTermination(ctx, params, configs)
	} else {
		tflog.Debug(ctx, "[resource] running as Executor")
		result, err = exec.Execute(ctx, params, configs)
	}
	if err != nil {
		diag.AddError(
			"Unable to read instance",
			fmt.Sprintf("Service returned with error: %v", err),
		)
		return nil
	}

	resultWithValue, ok := core.ResultAs[core.ResultWithValue](result)
	if !ok {
		diag.AddError(
			"Operation output mismatch",
			fmt.Sprintf("result has no value %+v", result),
		)
		return nil
	}

	/* TODO:
	if err := validateResult(diag, result); err != nil {
		return
	}
	*/
	_ = validateResult(diag, resultWithValue) // just ignore errors for now

	resultMap, ok := castToMap(resultWithValue, diag)
	if !ok {
		return nil
	}

	tflog.Debug(ctx, fmt.Sprintf("[resource] received new %s resource information: %#v", r.name, resultMap))
	return resultMap
}

func (r *MgcResource) applyStateAfter(exec core.Executor, params map[string]any, configs map[string]any, result core.ResultWithValue, ctx context.Context, tfState *tfsdk.State, diag *diag.Diagnostics) {
	var resultMap map[string]any
	resultSchema := exec.ResultSchema()

	/* TODO:
	if err := validateResult(diag, result); err != nil {
		return
	}
	*/
	_ = validateResult(diag, result) // just ignore errors for now

	if checkSimilarJsonSchemas(resultSchema, r.read.ResultSchema()) {
		var ok bool
		if resultMap, ok = castToMap(result, diag); !ok {
			return
		}
	} else {
		// TODO: when we implement links this will go away as it will get internally
		// https://github.com/profusion/magalu/issues/215
		mgcState := params
		if resultSchema.Type == "object" {
			hasAllProps := true
			for k := range r.read.ParametersSchema().Properties {
				if _, ok := resultSchema.Properties[k]; !ok {
					hasAllProps = false
					break
				}
			}
			if hasAllProps {
				var ok bool
				if mgcState, ok = castToMap(result, diag); !ok {
					return
				}
			}
		}

		resultMap = r.readResource(ctx, mgcState, configs, diag)
		if diag.HasError() {
			return
		}
	}

	r.applyMgcOutputMap(resultMap, ctx, tfState, diag)
}

func getConfigs(schema *core.Schema) map[string]core.Value {
	result := map[string]core.Value{}
	for propName, propRef := range schema.Properties {
		prop := (*core.Schema)(propRef.Value)
		if prop.Default != nil {
			result[propName] = prop.Default
		}
	}
	return result
}

func (r *MgcResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Make request
	configs := getConfigs(r.create.ConfigsSchema())
	params := r.readMgcMap(r.create.ParametersSchema(), ctx, tfsdk.State(req.Plan), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("[resource] creating `%s` - request info with params: %+v and configs: %+v", r.name, params, configs))
	ctx = r.sdk.WrapContext(ctx)
	exec := r.create
	result, err := exec.Execute(ctx, params, configs)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create resource",
			fmt.Sprintf("Service returned with error: %v", err),
		)
		return
	}

	resultWithValue, ok := core.ResultAs[core.ResultWithValue](result)
	if !ok {
		resp.Diagnostics.AddError(
			"Operation output mismatch",
			fmt.Sprintf("result has no value %+v", result),
		)
		return
	}

	r.applyStateAfter(exec, params, configs, resultWithValue, ctx, &resp.State, &resp.Diagnostics)

	// We must apply the input parameters in the state
	// BE CAREFUL: Don't apply Plan.Raw values into the State they might be Unknown! State only handles Known/Null values.
	r.applyMgcInputMap(params, ctx, &resp.State, &resp.Diagnostics)
	tflog.Info(ctx, fmt.Sprintf("[resource] created a %s resource", r.name))

}

func (r *MgcResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, fmt.Sprintf("[resource] reading `%s`", r.name))

	// Make request
	configs := getConfigs(r.read.ConfigsSchema())
	params := r.readMgcMap(r.read.ParametersSchema(), ctx, req.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = r.sdk.WrapContext(ctx)
	resultMap := r.readResource(ctx, params, configs, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.applyMgcOutputMap(resultMap, ctx, &resp.State, &resp.Diagnostics)
}

func (r *MgcResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	configs := getConfigs(r.update.ConfigsSchema())
	params := r.readMgcMap(r.update.ParametersSchema(), ctx, tfsdk.State(req.Plan), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = r.sdk.WrapContext(ctx)
	exec := r.update
	result, err := exec.Execute(ctx, params, configs)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update instance",
			fmt.Sprintf("Service returned with error: %v", err),
		)
		return
	}

	resultWithValue, ok := core.ResultAs[core.ResultWithValue](result)
	if !ok {
		resp.Diagnostics.AddError(
			"Operation output mismatch",
			fmt.Sprintf("result has no value %+v", result),
		)
		return
	}

	r.applyStateAfter(exec, params, configs, resultWithValue, ctx, &resp.State, &resp.Diagnostics)
}

func (r *MgcResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	configs := getConfigs(r.delete.ConfigsSchema())
	params := r.readMgcMap(r.delete.ParametersSchema(), ctx, tfsdk.State(req.State), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = r.sdk.WrapContext(ctx)
	exec := r.delete
	_, err := exec.Execute(ctx, params, configs)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete resource",
			fmt.Sprintf("Removing resource request returned with error: %v", err),
		)
		return
	}
}

func (r *MgcResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func validateResult(d *diag.Diagnostics, result core.ResultWithValue) error {
	err := result.ValidateSchema()
	if err != nil {
		d.AddWarning(
			"Operation output mismatch",
			fmt.Sprintf("Result has invalid structure: %v", err),
		)
	}
	return err
}
