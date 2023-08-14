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
	sdk      *mgcSdk.Sdk
	name     string
	group    mgcSdk.Grouper // TODO: is this needed?
	create   mgcSdk.Executor
	read     mgcSdk.Executor
	update   mgcSdk.Executor
	delete   mgcSdk.Executor
	atc      *map[string]attrConstraints
	tfschema *schema.Schema
}

type attrConstraints struct {
	isRequired                 bool
	isOptional                 bool
	isComputed                 bool
	useStateForUnknown         bool
	requiresReplaceWhenChanged bool
}

type attrValues struct {
	readResult   *mgcSdk.Schema
	create       *mgcSdk.Schema
	createResult *mgcSdk.Schema
	update       *mgcSdk.Schema
}

// TODO: remove once we translate directly from mgcSdk.Schema
type VirtualMachineResourceModel struct {
	Id       types.String `tfsdk:"id"`           // json:"id,omitempty"`
	Name     types.String `tfsdk:"name"`         // json:"name"`
	Type     types.String `tfsdk:"type"`         // json:"type"`
	Image    types.String `tfsdk:"image"`        // json:"image"`
	SSHKey   types.String `tfsdk:"key_name"`     // json:"key_name"`
	AllocFip types.Bool   `tfsdk:"allocate_fip"` // json:"allocate_fip"
	UserData types.String `tfsdk:"user_data"`    // json:"user_data"
	// Net      types.List   `tfsdk:"network_interfaces"` // json:"network_interfaces"
	Zone   types.String `tfsdk:"availability_zone"` // json:"availability_zone"
	Status types.String `tfsdk:"status"`            // json:"status"`
}

func (r *MgcResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.name
}

func (r *MgcResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// TODO: Handle nullable values
	// TODO: Move similar logic to another function
	if r.tfschema != nil {
		resp.Schema = *r.tfschema
		return
	}
	atc := map[string]attrConstraints{}

	rrrs := r.read.ResultSchema()
	rcps := r.create.ParametersSchema()
	rcrs := r.create.ResultSchema()
	rups := r.update.ParametersSchema()

	// Those structures work as a set, except for creationAttr which defined if
	// the attr is required or not
	allAttr := map[string]*attrValues{}

	tflog.Debug(ctx, fmt.Sprintf("[resource] generating schema for `%s`", r.name))

	createAttrValues := map[string](*mgcSdk.Schema){}
	createAttr := getRequiredOperationAttrs(ctx, r.name, rcps, &createAttrValues)

	createResultAttrValues := map[string](*mgcSdk.Schema){}
	createResultAttr := getOperationAttrs(ctx, r.name, rcrs, &createResultAttrValues)

	readResultAttrValues := map[string](*mgcSdk.Schema){}
	readResultAttr := getOperationAttrs(ctx, r.name, rrrs, &readResultAttrValues)

	updateAttrValues := map[string](*mgcSdk.Schema){}
	updateAttr := getOperationAttrs(ctx, r.name, rups, &updateAttrValues)

	// Attribute constraints rules:
	// - If on creation and update the user can update in-place
	// - If only on creation result it is computed
	// - If only in creation it can be optional or required and the user must replace when updating
	// - If not on creation, but on creation result and update it is optional and computed
	// - If only on update, it is optional and computed
	for k := range allAttr {
		_, rr := (*readResultAttr)[k]
		_, cr := (*createResultAttr)[k]
		req, c := (*createAttr)[k]
		_, up := (*updateAttr)[k]

		if c {
			atc[k] = attrConstraints{
				isRequired:                 req,
				isOptional:                 !req,
				isComputed:                 !req && rr, // If not required and present in read it can be computed
				useStateForUnknown:         false,
				requiresReplaceWhenChanged: !up,
			}
		} else if cr {
			// Here the order matters since for example ID exists in the Update but
			// we should only consider it if there is no return in the create
			// function
			atc[k] = attrConstraints{
				isRequired:                 false,
				isOptional:                 false,
				isComputed:                 true,
				useStateForUnknown:         true,
				requiresReplaceWhenChanged: false, // This one is useless in this case
			}
		} else if up {
			atc[k] = attrConstraints{
				isRequired:                 false,
				isOptional:                 true,
				isComputed:                 true,
				useStateForUnknown:         true,
				requiresReplaceWhenChanged: false,
			}
		} else if rr {
			atc[k] = attrConstraints{
				isRequired:                 false,
				isOptional:                 false,
				isComputed:                 true,
				useStateForUnknown:         true,
				requiresReplaceWhenChanged: false, // This one is useless in this case
			}
		} else {
			// TODO: Validate if there is other cases
			resp.Diagnostics.AddError(
				"[resource] unknown attribute constraints",
				fmt.Sprintf("attribute `%s` - read result: %t - create: %t - create result: %t - update: %t", k, rr, c, cr, up),
			)
			return
		}

		at := atc[k]
		tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: attribute `%s` info - %+v", r.name, k, at))
	}
	r.atc = &atc

	tfs := schema.Schema{}
	tfs.MarkdownDescription = r.name
	tfs.Attributes = map[string]schema.Attribute{}
	for k, v := range atc {
		attr := sdkToTerraformAttribute(ctx, *allAttr[k], v, resp.Diagnostics)
		if attr == nil {
			// TODO: Error
			tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: unable to identify attribute `%s`", r.name, k))
			continue
		}
		tfs.Attributes[k] = attr
		tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: inserted attribute `%s` to the schema", r.name, k))
	}

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
	data.Zone = types.StringValue(resultData["availability_zone"].(string))
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
