package provider

import (
	"context"

	"slices"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
	mgcSdk "magalu.cloud/sdk"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &MgcResource{}
var _ resource.ResourceWithImportState = &MgcResource{}

// MgcResource defines the resource implementation.
type MgcResource struct {
	sdk               *mgcSdk.Sdk
	name              string
	description       string
	group             mgcSdk.Grouper // TODO: is this needed?
	create            mgcSdk.Executor
	read              mgcSdk.Executor
	update            mgcSdk.Executor
	delete            mgcSdk.Executor
	inputAttrInfoMap  resAttrInfoMap
	outputAttrInfoMap resAttrInfoMap
	splitAttributes   []splitResAttribute
	tfschema          *schema.Schema
}

// BEGIN: tfSchemaHandler implementation

func (r *MgcResource) Name() string {
	return r.name
}

func (r *MgcResource) getCreateParamsModifiers(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	k := string(mgcName)
	isRequired := slices.Contains(mgcSchema.Required, k)
	isComputed := !isRequired
	if isComputed {
		readSchema := r.read.ResultSchema().Properties[k]
		if readSchema == nil {
			isComputed = false
		} else {
			// If not required and present in read it can be compute
			isComputed = mgcSchemaPkg.CheckSimilarJsonSchemas((*core.Schema)(readSchema.Value), (*core.Schema)(mgcSchema.Properties[string(mgcName)].Value))
		}
	}

	return attributeModifiers{
		isRequired:                 isRequired,
		isOptional:                 !isRequired,
		isComputed:                 isComputed,
		useStateForUnknown:         false,
		requiresReplaceWhenChanged: r.update.ParametersSchema().Properties[k] == nil,
		getChildModifiers:          getInputChildModifiers,
	}
}

func (r *MgcResource) getUpdateParamsModifiers(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	k := string(mgcName)
	isComputed := r.create.ResultSchema().Properties[k] != nil
	required := slices.Contains(mgcSchema.Required, k)

	return attributeModifiers{
		isRequired:                 required && !isComputed,
		isOptional:                 !required && !isComputed,
		isComputed:                 !required || isComputed,
		useStateForUnknown:         true,
		requiresReplaceWhenChanged: false,
		getChildModifiers:          getInputChildModifiers,
	}
}

func (r *MgcResource) getDeleteParamsModifiers(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	isComputed := r.create.ResultSchema().Properties[string(mgcName)] != nil

	// All Delete parameters need to be optional, since they're not returned by the server when creating the resources,
	// and thus would produce "inconsistent results" in Terraform. Since we calculate the 'Update' parameters first,
	// all strictly necessary parameters for deletion (like the resource ID) will already be computed correctly (in the Update
	// parameters)
	return attributeModifiers{
		isRequired:                 false,
		isOptional:                 true,
		isComputed:                 isComputed,
		useStateForUnknown:         true,
		requiresReplaceWhenChanged: false,
		getChildModifiers:          getInputChildModifiers,
	}
}

func (r *MgcResource) ReadInputAttributes(ctx context.Context) diag.Diagnostics {
	ctx = tflog.SubsystemSetField(ctx, schemaGenSubsystem, resourceNameField, r.name)
	d := diag.Diagnostics{}
	if len(r.inputAttrInfoMap) != 0 {
		return d
	}
	tflog.SubsystemDebug(ctx, schemaGenSubsystem, "reading input attributes")

	input := resAttrInfoMap{}
	err := addMgcSchemaAttributes(
		input,
		r.create.ParametersSchema(),
		r.getCreateParamsModifiers,
		ctx,
	)
	if err != nil {
		d.AddError("could not create TF input attributes", err.Error())
		return d
	}

	err = addMgcSchemaAttributes(
		input,
		r.update.ParametersSchema(),
		r.getUpdateParamsModifiers,
		ctx,
	)
	if err != nil {
		d.AddError("could not create TF input attributes", err.Error())
		return d
	}

	err = addMgcSchemaAttributes(
		input,
		r.delete.ParametersSchema(),
		r.getDeleteParamsModifiers,
		ctx,
	)
	if err != nil {
		d.AddError("could not create TF input attributes", err.Error())
		return d
	}

	r.inputAttrInfoMap = input
	return d
}

func (r *MgcResource) ReadOutputAttributes(ctx context.Context) diag.Diagnostics {
	ctx = tflog.SubsystemSetField(ctx, schemaGenSubsystem, resourceNameField, r.name)
	d := diag.Diagnostics{}
	if len(r.outputAttrInfoMap) != 0 {
		return d
	}
	tflog.SubsystemDebug(ctx, schemaGenSubsystem, "reading output attributes")

	output := resAttrInfoMap{}
	err := addMgcSchemaAttributes(
		output,
		r.create.ResultSchema(),
		getResultModifiers,
		ctx,
	)
	if err != nil {
		d.AddError("could not create TF output attributes from create request", err.Error())
		return d
	}
	err = addMgcSchemaAttributes(
		output,
		r.read.ResultSchema(),
		getResultModifiers,
		ctx,
	)
	if err != nil {
		d.AddError("could not create TF output attributes from read request", err.Error())
		return d
	}

	r.outputAttrInfoMap = output
	return d
}

func (r *MgcResource) InputAttributes() resAttrInfoMap {
	return r.inputAttrInfoMap
}

func (r *MgcResource) OutputAttributes() resAttrInfoMap {
	return r.outputAttrInfoMap
}

func (r *MgcResource) AppendSplitAttribute(split splitResAttribute) {
	if r.splitAttributes == nil {
		r.splitAttributes = []splitResAttribute{}
	}
	r.splitAttributes = append(r.splitAttributes, split)
}

var _ tfSchemaHandler = (*MgcResource)(nil)

// END: tfSchemaHandler implementation

// BEGIN: tfStateHandler implementation

func (r *MgcResource) SplitAttributes() []splitResAttribute {
	return r.splitAttributes
}

func (r *MgcResource) TFSchema() *schema.Schema {
	return r.tfschema
}

func (r *MgcResource) ReadResultSchema() *mgcSdk.Schema {
	return r.read.ResultSchema()
}

var _ tfStateHandler = (*MgcResource)(nil)

// END: tfStateHandler implementation

// BEGIN: Resource implementation

func (r *MgcResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.name
}

func (r *MgcResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	ctx = tflog.SetField(ctx, rpcField, "schema")
	ctx = tflog.SetField(ctx, resourceNameField, r.name)
	tflog.Debug(ctx, "generating schema")

	if r.tfschema == nil {
		tfs, d := generateTFSchema(r, ctx)
		resp.Diagnostics.Append(d...)
		if d.HasError() {
			return
		}

		tfs.MarkdownDescription = r.description
		r.tfschema = &tfs
	}

	resp.Schema = *r.tfschema
}

func (r *MgcResource) performOperation(ctx context.Context, exec core.Executor, inState tfsdk.State, outState *tfsdk.State, diag *diag.Diagnostics) {
	ctx = r.sdk.WrapContext(ctx)

	configs := getConfigs(exec.ConfigsSchema())
	params := readMgcMapSchemaFromTFState(r, exec.ParametersSchema(), ctx, inState, diag)
	if diag.HasError() {
		return
	}

	result := execute(r.name, ctx, exec, params, configs, diag)
	if diag.HasError() {
		return
	}

	if outState != nil {
		applyStateAfter(r, result, r.read, ctx, outState, diag)
	}
}

func (r *MgcResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	ctx = tflog.SetField(ctx, rpcField, "create")
	ctx = tflog.SetField(ctx, resourceNameField, r.name)
	r.performOperation(ctx, r.create, tfsdk.State(req.Plan), &resp.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "resource created")
}

func (r *MgcResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx = tflog.SetField(ctx, rpcField, "read")
	ctx = tflog.SetField(ctx, resourceNameField, r.name)
	r.performOperation(ctx, r.read, req.State, &resp.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "resource read")
}

func (r *MgcResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	ctx = tflog.SetField(ctx, rpcField, "update")
	ctx = tflog.SetField(ctx, resourceNameField, r.name)
	r.performOperation(ctx, r.update, tfsdk.State(req.Plan), &resp.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "resource updated")
}

func (r *MgcResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	ctx = tflog.SetField(ctx, rpcField, "delete")
	ctx = tflog.SetField(ctx, resourceNameField, r.name)
	r.performOperation(ctx, r.delete, req.State, nil, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "resource deleted")
}

func (r *MgcResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx = tflog.SetField(ctx, rpcField, "import-state")
	ctx = tflog.SetField(ctx, resourceNameField, r.name)
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// END: Resource implemenation
