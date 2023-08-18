package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	inputAttr  map[string]*attribute
	outputAttr map[string]*attribute
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

	if r.tfschema != nil {
		resp.Schema = *r.tfschema
		return
	}

	tfsa, d := r.generateTFAttributes(ctx)
	resp.Diagnostics.Append(d...)

	tfs := schema.Schema{}
	tfs.MarkdownDescription = r.name
	tfs.Attributes = *tfsa

	r.tfschema = &tfs
	resp.Schema = tfs
}

func (r *MgcResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *VirtualMachineResourceModel

	// TODO: remove once we translate directly from mgcSdk.Schema
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Make request
	params := map[string]any{
		"name":     data.Name.ValueString(),
		"type":     data.Type.ValueString(),
		"image":    data.DesiredImage.ValueString(),
		"key_name": data.SSHKey.ValueString(),
	}
	// TODO: read from req.Config
	configs := map[string]any{}
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

	resultMap, ok := result.(map[string]any)
	if !ok {
		resp.Diagnostics.AddError(
			"Operation output mismatch",
			fmt.Sprintf("Unable to convert %v to map.", result),
		)
		return
	}

	id, ok := resultMap["id"].(string)
	if !ok {
		resp.Diagnostics.AddWarning(
			"Operation output mismatch",
			fmt.Sprintf("Unable to convert %v to string.", resultMap["id"]),
		)
	}

	data.Id = types.StringValue(id)
	tflog.Trace(ctx, "created a virtual-machine resource with id %s")

	// TODO: set resp.State directly from resultMap, without going to `data`(Model)
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MgcResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "reading vm instance resource information")
	var data *VirtualMachineResourceModel

	// TODO: remove once we translate directly from mgcSdk.Schema
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: generate params from State, must ensure names match between ResultSchema + ParametersSchema

	// Make request
	tflog.Info(ctx, fmt.Sprintf("retrieving `instance` information for ID %s", data.Id.ValueString()))
	params := map[string]any{
		"id": data.Id.ValueString(),
	}

	result, err := r.read.Execute(r.sdk.WrapContext(ctx), params, map[string]any{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get instance",
			fmt.Sprintf("Fetching information for instance %v returned with error: %v", data.Id, err),
		)
		return
	}

	/* TODO:
	if err := validateResult(resp.Diagnostics, r.create, result); err != nil {
		return
	}
	*/
	_ = validateResult(resp.Diagnostics, r.create, result) // just ignore errors for now

	resultData, ok := result.(map[string]any)
	if !ok {
		resp.Diagnostics.AddError(
			"Operation output mismatch",
			fmt.Sprintf("Unable to convert %v to map.", result),
		)
		return
	}

	data.Id = types.StringValue(resultData["id"].(string))
	data.InstanceID = types.StringValue(resultData["instance_id"].(string))
	data.Name = types.StringValue(resultData["name"].(string))
	data.Type = types.StringValue(resultData["instance_type"].(map[string]any)["name"].(string))
	data.SSHKey = types.StringValue(resultData["key_name"].(string))
	data.Zone = types.StringValue(resultData["availability_zone"].(string))
	data.CurrentStatus = types.StringValue(strings.ToLower(resultData["status"].(string)))

	// TODO: set resp.State directly from resultMap, without going to `data`(Model)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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
