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

func (r *MgcResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	conv := newTFStateConverter(ctx, &resp.Diagnostics, r.tfschema)

	// Make request
	configs := map[string]any{}
	iatinfo := attribute{
		tfName:     "inputSchemasInfo",
		mgcName:    "inputSchemasInfo",
		mgcSchema:  r.create.ParametersSchema(),
		attributes: r.inputAttr,
	}
	oatinfo := attribute{
		tfName:     "outputSchemasInfo",
		mgcName:    "outputSchemasInfo",
		mgcSchema:  r.create.ResultSchema(),
		attributes: r.outputAttr,
	}

	// Create initial state from create params and update params
	// TODO: Find a better way to send the filter flags
	tfStateMap := map[string]any{}
	mgcCreateMap := conv.toMgcSchemaMap(r.create.ParametersSchema(), &iatinfo, req.Plan.Raw, true, false)
	conv.mgcKeysToStateKeys(&iatinfo, mgcCreateMap, tfStateMap)
	mgcUpdateStateMap := conv.toMgcSchemaMap(r.update.ParametersSchema(), &iatinfo, req.Plan.Raw, true, false)
	conv.mgcKeysToStateKeys(&iatinfo, mgcUpdateStateMap, tfStateMap)

	params := conv.toMgcSchemaMap(r.create.ParametersSchema(), &iatinfo, req.Plan.Raw, true, true)
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

	// Add create result information to state
	conv.mgcKeysToStateKeys(&oatinfo, mgcCreateResultMap, tfStateMap)

	tflog.Info(ctx, "[resource] created a virtual-machine resource with id %s")

	// TODO: Wait until the desired status is achieved - Remove sleep timer
	time.Sleep(time.Second * 20)

	// Read param elements from create result
	params = map[string]any{}
	for k := range r.read.ParametersSchema().Properties {
		params[k] = mgcCreateResultMap[k]
	}
	tflog.Debug(ctx, "[resource] reading new virtual-machine resource with id %s")
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

	// Update state with read result
	conv.mgcKeysToStateKeys(&oatinfo, mgcReadResultMap, tfStateMap)

	// Create a tf state value from the state map
	tfStateValue := conv.fromMap(tfStateMap)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.State = tfsdk.State{
		Raw:    *tfStateValue,
		Schema: resp.State.Schema,
	}
}

func (r *MgcResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, fmt.Sprintf("[resource] reading `%s`", r.name))

	conv := newTFStateConverter(ctx, &resp.Diagnostics, r.tfschema)
	// TODO: Convert entire state to a map

	// Make request
	atinfo := attribute{
		tfName:     "schema",
		attributes: r.inputAttr,
	}
	params := conv.toMgcSchemaMap(r.read.ParametersSchema(), &atinfo, req.State.Raw, true, true)
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

	// TODO: Update current state
}

func (r *MgcResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *VirtualMachineResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"id":     data.Id.ValueString(),
		"status": data.DesiredStatus.ValueString(),
	}
	_, err := r.update.Execute(r.sdk.WrapContext(ctx), params, map[string]any{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get instance",
			fmt.Sprintf("Fetching information for instance %v returned with error: %v", data.Id, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MgcResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *VirtualMachineResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := map[string]any{
		"id": data.Id.ValueString(),
	}
	_, err := r.delete.Execute(r.sdk.WrapContext(ctx), params, map[string]any{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get instance",
			fmt.Sprintf("Fetching information for instance %v returned with error: %v", data.Id, err),
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
