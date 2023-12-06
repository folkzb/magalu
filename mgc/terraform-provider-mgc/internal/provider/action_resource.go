package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"magalu.cloud/core"
	mgcSdk "magalu.cloud/sdk"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &MgcActionResource{}
var _ resource.ResourceWithImportState = &MgcActionResource{}

// MgcResource defines the resource implementation.
type MgcActionResource struct {
	sdk               *mgcSdk.Sdk
	name              string
	readOwner         mgcSdk.Executor
	create            mgcSdk.Linker
	read              mgcSdk.Linker
	update            mgcSdk.Linker // TODO: Will action resources need/have updates?
	delete            mgcSdk.Linker
	inputAttrInfoMap  resAttrInfoMap
	outputAttrInfoMap resAttrInfoMap
	splitAttributes   []splitResAttribute
	tfschema          *schema.Schema
}

// BEGIN: tfSchemaHandler implementation

func (r *MgcActionResource) Name() string {
	return r.name
}

func (r *MgcActionResource) Description() string {
	return r.name
}

func (r *MgcActionResource) getReadParamsModifiers(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	return attributeModifiers{
		isRequired:                 true,
		isOptional:                 false,
		isComputed:                 false,
		useStateForUnknown:         true,
		requiresReplaceWhenChanged: true,
		getChildModifiers:          getInputChildModifiers,
	}
}

func (r *MgcActionResource) getDeleteParamsModifiers(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	// For now we consider all delete params as optionals, we need to think a way for the user to define
	// required delete params
	return attributeModifiers{
		isRequired:                 false,
		isOptional:                 true,
		isComputed:                 false,
		useStateForUnknown:         true,
		requiresReplaceWhenChanged: true,
		getChildModifiers:          getInputChildModifiers,
	}
}

func (r *MgcActionResource) InputAttrInfoMap(ctx context.Context, d *diag.Diagnostics) resAttrInfoMap {
	if r.inputAttrInfoMap == nil {
		r.inputAttrInfoMap = generateResAttrInfoMap(ctx, r.name,
			[]resAttrInfoGenMetadata{
				{r.create.AdditionalParametersSchema(), r.getReadParamsModifiers},
				{r.readOwner.ParametersSchema(), r.getReadParamsModifiers},
				{r.delete.AdditionalParametersSchema(), r.getDeleteParamsModifiers},
			}, d,
		)
	}
	return r.inputAttrInfoMap
}

func (r *MgcActionResource) OutputAttrInfoMap(ctx context.Context, d *diag.Diagnostics) resAttrInfoMap {
	if r.outputAttrInfoMap == nil {
		r.outputAttrInfoMap = generateResAttrInfoMap(ctx, r.name,
			[]resAttrInfoGenMetadata{
				{r.create.ResultSchema(), getResultModifiers},
				{r.read.ResultSchema(), getResultModifiers},
			}, d,
		)
	}
	return r.outputAttrInfoMap
}

func (r *MgcActionResource) AppendSplitAttribute(split splitResAttribute) {
	if r.splitAttributes == nil {
		r.splitAttributes = []splitResAttribute{}
	}
	r.splitAttributes = append(r.splitAttributes, split)
}

var _ tfSchemaHandler = (*MgcActionResource)(nil)

// END: tfSchemaHandler implementation

// BEGIN: tfStateHandler implementation

func (r *MgcActionResource) TFSchema() *schema.Schema {
	return r.tfschema
}

func (r *MgcActionResource) SplitAttributes() []splitResAttribute {
	return r.splitAttributes
}

func (r *MgcActionResource) ReadResultSchema() *mgcSdk.Schema {
	return r.read.ResultSchema()
}

var _ tfStateHandler = (*MgcActionResource)(nil)

// END: tfStateHandler implementation

// BEGIN: Resource implementation

func (r *MgcActionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.name
}

func (r *MgcActionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	if r.tfschema == nil {
		ctx = tflog.SetField(ctx, actionResourceNameField, r.name)
		tfs := generateTFSchema(r, ctx, &resp.Diagnostics)
		r.tfschema = &tfs
	}
	resp.Schema = *r.tfschema
}

func (r *MgcActionResource) performOwnerRead(ctx context.Context, tfState tfsdk.State, d *diag.Diagnostics) core.ResultWithValue {
	configs := getConfigs(r.readOwner.ConfigsSchema())
	params := readMgcMapSchemaFromTFState(r, r.readOwner.ParametersSchema(), ctx, tfState, d)
	if d.HasError() {
		return nil
	}

	return execute(r.name, ctx, r.readOwner, params, configs, d)
}

func (r *MgcActionResource) performLinkOperation(ctx context.Context, link core.Linker, inState tfsdk.State, outState *tfsdk.State, diag *diag.Diagnostics) {
	ctx = r.sdk.WrapContext(ctx)

	ownerResult := r.performOwnerRead(ctx, inState, diag)
	if diag.HasError() {
		return
	}

	configs := getConfigs(link.AdditionalConfigsSchema())
	params := readMgcMapSchemaFromTFState(r, link.AdditionalParametersSchema(), ctx, inState, diag)
	if diag.HasError() {
		return
	}

	linkExec, err := link.CreateExecutor(ownerResult)
	if err != nil {
		diag.AddError("error when creating link executor", err.Error())
		return
	}
	result := execute(r.name, ctx, linkExec, params, configs, diag)
	if diag.HasError() {
		return
	}
	if outState != nil {
		applyStateAfter(r, result, nil, ctx, outState, diag)
	}
}

func (r *MgcActionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	ctx = tflog.SetField(ctx, rpcField, "create")
	ctx = tflog.SetField(ctx, actionResourceNameField, r.name)
	r.performLinkOperation(ctx, r.create, tfsdk.State(req.Plan), &resp.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "resource created")
}

func (r *MgcActionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx = tflog.SetField(ctx, rpcField, "read")
	ctx = tflog.SetField(ctx, actionResourceNameField, r.name)
	r.performLinkOperation(ctx, r.read, req.State, &resp.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		// When reading fails, that means that the resource was most likely altered outside of terraform.
		resp.Diagnostics.AddError("reading the resource failed", "was the resource altered outside of terraform?")
		return
	}
	tflog.Info(ctx, "resource read")
}

// Update will most likely never be called, as we always require replace when changed
func (r *MgcActionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	ctx = tflog.SetField(ctx, rpcField, "update")
	ctx = tflog.SetField(ctx, actionResourceNameField, r.name)
	if r.update == nil {
		resp.Diagnostics.AddError(
			"no 'update' operation was provided",
			fmt.Sprintf("action resource %q doesn't have an update operation to run", r.name),
		)
		return
	}
	r.performLinkOperation(ctx, r.update, tfsdk.State(req.Plan), &resp.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "resource updated")
}

func (r *MgcActionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	ctx = tflog.SetField(ctx, rpcField, "delete")
	ctx = tflog.SetField(ctx, actionResourceNameField, r.name)
	r.performLinkOperation(ctx, r.delete, req.State, nil, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "resource deleted")
}

func (r *MgcActionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx = tflog.SetField(ctx, rpcField, "import-state")
	ctx = tflog.SetField(ctx, actionResourceNameField, r.name)
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ resource.Resource = (*MgcActionResource)(nil)

// END: Resource implementation
