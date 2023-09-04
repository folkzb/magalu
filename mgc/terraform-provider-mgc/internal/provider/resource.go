package provider

import (
	"context"
	"fmt"
	"time"

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

func (r *MgcResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Make request
	configs := map[string]any{}
	params := r.readMgcMap(r.create.ParametersSchema(), ctx, tfsdk.State(req.Plan), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("[resource] creating `%s` - request info with params: %+v", r.name, params))
	result, err := r.create.Execute(r.sdk.WrapContext(ctx), params, configs)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create instance",
			fmt.Sprintf("Service returned with error: %v", err),
		)
		return
	}

	/* TODO:
	if err := validateResult(resp.Diagnostics, r.create, result); err != nil {
		return
	}
	*/
	_ = validateResult(resp.Diagnostics, r.create, result) // just ignore errors for now

	mgcCreateResultMap, ok := result.(map[string]any)
	if !ok {
		resp.Diagnostics.AddError(
			"Operation output mismatch",
			fmt.Sprintf("Unable to convert %v to map.", result),
		)
		return
	}

	// We must apply the input parameters in the state
	// BE CAREFUL: Don't apply Plan.Raw values into the State they might be Unknown! State only handles Known/Null values.
	r.applyMgcInputMap(params, ctx, &resp.State, &resp.Diagnostics)
	tflog.Info(ctx, "[resource] created a virtual-machine resource")

	var resultMap map[string]any
	if checkSimilarJsonSchemas(r.create.ResultSchema(), r.read.ResultSchema()) {
		resultMap = mgcCreateResultMap
	} else {
		// TODO: Wait until the desired status is achieved - Remove sleep timer
		time.Sleep(time.Minute)

		// TODO: this is going away when we implement links
		// see: https://github.com/profusion/magalu/issues/215
		// Read param elements from create result
		params = map[string]any{}
		for k := range r.read.ParametersSchema().Properties {
			params[k] = mgcCreateResultMap[k]
		}
		tflog.Debug(ctx, "[resource] reading new virtual-machine resource")
		result, err = r.read.Execute(r.sdk.WrapContext(ctx), params, configs)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to create instance",
				fmt.Sprintf("Service returned with error: %v", err),
			)
			return
		}
		_ = validateResult(resp.Diagnostics, r.create, result) // just ignore errors for now

		mgcReadResultMap, ok := result.(map[string]any)
		if !ok {
			resp.Diagnostics.AddError(
				"Operation output mismatch",
				fmt.Sprintf("Unable to convert %v to map.", result),
			)
			return
		}
		tflog.Debug(ctx, fmt.Sprintf("[resource] received new virtual-machine resource information: %#v", mgcReadResultMap))
		resultMap = mgcReadResultMap
	}

	r.applyMgcOutputMap(resultMap, ctx, &resp.State, &resp.Diagnostics)
}

func (r *MgcResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, fmt.Sprintf("[resource] reading `%s`", r.name))

	// Make request
	params := r.readMgcMap(r.read.ParametersSchema(), ctx, req.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("[resource] reading `%s` - request info with params: %+v", r.name, params))
	result, err := r.read.Execute(r.sdk.WrapContext(ctx), params, map[string]any{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get resource",
			fmt.Sprintf("[resource] information for `%s` returned with error: %v", r.name, err),
		)
		return
	}

	/* TODO:
	if err := validateResult(resp.Diagnostics, r.create, result); err != nil {
		return
	}
	*/
	_ = validateResult(resp.Diagnostics, r.create, result) // just ignore errors for now

	resultMap, ok := result.(map[string]any)
	if !ok {
		resp.Diagnostics.AddError(
			"Operation output mismatch",
			fmt.Sprintf("Unable to convert %v to map.", result),
		)
		return
	}

	r.applyMgcOutputMap(resultMap, ctx, &resp.State, &resp.Diagnostics)
}

func (r *MgcResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	params := r.readMgcMap(r.update.ParametersSchema(), ctx, tfsdk.State(req.Plan), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.update.Execute(r.sdk.WrapContext(ctx), params, map[string]any{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get instance",
			fmt.Sprintf("Fetching information for instance returned with error: %v", err),
		)
		return
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		resp.Diagnostics.AddError(
			"Operation output mismatch",
			fmt.Sprintf("Unable to convert %v to map.", result),
		)
		return
	}

	r.applyMgcOutputMap(resultMap, ctx, &resp.State, &resp.Diagnostics)
}

func (r *MgcResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	params := r.readMgcMap(r.delete.ParametersSchema(), ctx, tfsdk.State(req.State), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.delete.Execute(r.sdk.WrapContext(ctx), params, map[string]any{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get instance",
			fmt.Sprintf("Fetching information for instance returned with error: %v", err),
		)
		return
	}
}

func (r *MgcResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func validateResult(d diag.Diagnostics, action core.Executor, result any) error {
	err := action.ResultSchema().VisitJSON(result)
	if err != nil {
		d.AddWarning(
			"Operation output mismatch",
			fmt.Sprintf("Result has invalid structure: %v", err),
		)
	}
	return err
}
