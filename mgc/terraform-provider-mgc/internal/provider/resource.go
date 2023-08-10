package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
	sdk    *mgcSdk.Sdk
	name   string
	group  mgcSdk.Grouper // TODO: is this needed?
	create mgcSdk.Executor
	read   mgcSdk.Executor
	update mgcSdk.Executor
	delete mgcSdk.Executor
}

// TODO: remove once we translate directly from mgcSdk.Schema
type VirtualMachineResourceModel struct {
	Id     types.String `tfsdk:"id"`       // json:"id,omitempty"`
	Name   types.String `tfsdk:"name"`     // json:"name"`
	Type   types.String `tfsdk:"type"`     // json:"type"`
	Image  types.String `tfsdk:"image"`    // json:"image"`
	SSHKey types.String `tfsdk:"key_name"` // json:"key_name"`
	Status types.String `tfsdk:"status"`   // json:"status"`
}

func (r *MgcResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.name
}

func (r *MgcResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// TODO: read from r.read.ResultSchema
	// TODO: how to detect computed? What's in the parameter? (ex: id), what else? (ie: status)
	resp.Schema = schema.Schema{
		MarkdownDescription: "Virtual Machine resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Instance id",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Instance name",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Instance type",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"image": schema.StringAttribute{
				MarkdownDescription: "OS image",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_name": schema.StringAttribute{
				MarkdownDescription: "SSH key",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Instance status",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("active"),
			},
		},
	}
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
		"image":    data.Image.ValueString(),
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
	data.Image = types.StringValue(resultData["image"].(map[string]any)["name"].(string))
	data.Name = types.StringValue(resultData["name"].(string))
	data.SSHKey = types.StringValue(resultData["key_name"].(string))
	data.Type = types.StringValue(resultData["instance_type"].(map[string]any)["name"].(string))
	data.Status = types.StringValue(strings.ToLower(resultData["status"].(string)))

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
		"status": data.Status.ValueString(),
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
