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

const (
	createResultKey = "create-result"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &MgcConnectionResource{}
var _ resource.ResourceWithImportState = &MgcConnectionResource{}

// MgcConnectionResource defines Connection Resources via Links. Conenction Resources aren't real resources
// themselves, they represent conenctions to be taken regarding another resource. For example, turning
// a resource instance on on off, modifying its status, etc.
type MgcConnectionResource struct {
	sdk         *mgcSdk.Sdk
	name        tfName
	description string
	create      mgcSdk.Executor
	read        mgcSdk.Linker
	update      mgcSdk.Linker // TODO: Will conenction resources need/have updates?
	delete      mgcSdk.Linker
	inputAttr   resAttrInfoMap
	outputAttr  resAttrInfoMap
	splitAttr   []splitResAttribute
	tfschema    *schema.Schema
}

func newMgcConnectionResource(
	ctx context.Context,
	sdk *mgcSdk.Sdk,
	name tfName,
	description string,
	connection mgcSdk.Executor,
	sourceDelete mgcSdk.Executor,
) (*MgcConnectionResource, error) {
	var read, update, delete mgcSdk.Linker
	for k, link := range connection.Links() {
		switch k {
		case "read":
			read = link
		case "update":
			update = link
		case "delete":
			delete = link
		}
	}

	if read == nil {
		return nil, fmt.Errorf("Connection Resource %q misses read", name)
	}
	if delete == nil {
		return nil, fmt.Errorf("Connection Resource %q misses delete", name)
	}
	if delete.ResultSchema() == sourceDelete.ResultSchema() {
		return nil, fmt.Errorf("Connection Resource %q's delete link targets the source resource deletion, not the connection deletion", name)
	}
	if update == nil {
		tflog.Warn(ctx, fmt.Sprintf("Connection Resource %s misses update operations", name))
		update = core.NewSimpleLink(connection, core.NoOpExecutor())
	}
	return &MgcConnectionResource{
		sdk:         sdk,
		name:        name,
		description: description,
		create:      connection,
		read:        read,
		update:      update,
		delete:      delete,
	}, nil
}

// BEGIN: tfSchemaHandler implementation

func (r *MgcConnectionResource) Name() string {
	return string(r.name)
}

func (r *MgcConnectionResource) Description() string {
	return r.description
}

func (r *MgcConnectionResource) getReadParamsModifiers(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	return attributeModifiers{
		isRequired:                 true,
		isOptional:                 false,
		isComputed:                 false,
		useStateForUnknown:         true,
		requiresReplaceWhenChanged: true,
		getChildModifiers:          getInputChildModifiers,
	}
}

func (r *MgcConnectionResource) getDeleteParamsModifiers(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	// TODO: For now we consider all delete params as optionals, we need to think a way for the user to define
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

func (r *MgcConnectionResource) InputAttrInfoMap(ctx context.Context, d *diag.Diagnostics) resAttrInfoMap {
	if r.inputAttr == nil {
		r.inputAttr = generateResAttrInfoMap(ctx, r.name,
			[]resAttrInfoGenMetadata{
				{r.create.ParametersSchema(), r.getReadParamsModifiers},
				{r.read.AdditionalParametersSchema(), r.getReadParamsModifiers},
				{r.delete.AdditionalParametersSchema(), r.getDeleteParamsModifiers},
			}, d,
		)
	}
	return r.inputAttr
}

func (r *MgcConnectionResource) OutputAttrInfoMap(ctx context.Context, d *diag.Diagnostics) resAttrInfoMap {
	if r.outputAttr == nil {
		r.outputAttr = generateResAttrInfoMap(ctx, r.name,
			[]resAttrInfoGenMetadata{
				{r.create.ResultSchema(), getResultModifiers},
				{r.read.ResultSchema(), getResultModifiers},
			}, d,
		)
	}
	return r.outputAttr
}

func (r *MgcConnectionResource) AppendSplitAttribute(split splitResAttribute) {
	if r.splitAttr == nil {
		r.splitAttr = []splitResAttribute{}
	}
	r.splitAttr = append(r.splitAttr, split)
}

var _ tfSchemaHandler = (*MgcConnectionResource)(nil)

// END: tfSchemaHandler implementation

// BEGIN: tfStateHandler implementation

func (r *MgcConnectionResource) TFSchema() *schema.Schema {
	return r.tfschema
}

func (r *MgcConnectionResource) SplitAttributes() []splitResAttribute {
	return r.splitAttr
}

func (r *MgcConnectionResource) ReadResultSchema() *mgcSdk.Schema {
	return r.read.ResultSchema()
}

var _ tfStateHandler = (*MgcConnectionResource)(nil)

// END: tfStateHandler implementation

// BEGIN: Resource implementation

func (r *MgcConnectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = string(r.name)
}

func (r *MgcConnectionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	if r.tfschema == nil {
		ctx = tflog.SetField(ctx, resourceNameField, r.name)
		tfs := generateTFSchema(r, ctx, &resp.Diagnostics)
		r.tfschema = &tfs
	}
	resp.Schema = *r.tfschema
}

func (r *MgcConnectionResource) performLinkOperation(
	ctx context.Context,
	link mgcSdk.Linker,
	originalResult mgcSdk.Result,
	inState tfsdk.State,
	outState *tfsdk.State,
	diag *diag.Diagnostics,
) {
	ctx = r.sdk.WrapContext(ctx)
	configs := getConfigs(ctx, link.AdditionalConfigsSchema())
	params := readMgcMapSchemaFromTFState(r, link.AdditionalParametersSchema(), ctx, inState, diag)
	if diag.HasError() {
		return
	}

	linkExec, err := link.CreateExecutor(originalResult)
	if err != nil {
		diag.AddError("error when creating link executor", err.Error())
		return
	}
	result := execute(r.name, ctx, linkExec, params, configs, diag)
	if diag.HasError() {
		return
	}
	applyStateAfter(r, result, nil, ctx, outState, diag)
}

func (r *MgcConnectionResource) performLinkOperationFromScratch(
	ctx context.Context,
	link mgcSdk.Linker,
	getPrivateStateKey func(context.Context, string) ([]byte, diag.Diagnostics),
	setPrivateStateKey func(context.Context, string, []byte) diag.Diagnostics,
	inState tfsdk.State,
	outState *tfsdk.State,
	diag *diag.Diagnostics,
) {
	createResultData, keyDiag := getPrivateStateKey(ctx, createResultKey)
	diag.Append(keyDiag...)
	if diag.HasError() {
		diag.AddError("unable to read creation result from Terraform state", "")
		return
	}

	tflog.Debug(ctx, "[connection-resource] about to decode creation result", map[string]any{"encoded result": string(createResultData)})
	createResult := r.create.EmptyResult()
	err := createResult.Decode(createResultData)
	if err != nil {
		diag.AddError("Failed to decode creation result", fmt.Sprintf("%v", err))
		return
	}

	keyDiag = setPrivateStateKey(ctx, createResultKey, createResultData)
	diag.Append(keyDiag...)

	r.performLinkOperation(ctx, link, createResult, inState, outState, diag)
}

func (r *MgcConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	ctx = tflog.SetField(ctx, rpcField, "create")
	ctx = tflog.SetField(ctx, connectionResourceNameField, r.name)
	ctx = r.sdk.WrapContext(ctx)

	configs := getConfigs(ctx, r.create.ConfigsSchema())
	params := readMgcMapSchemaFromTFState(r, r.create.ParametersSchema(), ctx, tfsdk.State(req.Plan), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	result := execute(r.name, ctx, r.create, params, configs, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	applyStateAfter(r, result, nil, ctx, &resp.State, &resp.Diagnostics)
	tflog.Info(ctx, "resource updated")

	resultEncoded, err := result.Encode()
	if err != nil {
		resp.Diagnostics.AddError(
			"failure to encode connection resource creation result",
			"Terraform wasn't able to encode the result of the creation process to save in its state. Creation was successful, but resource will be deleted, try again.",
		)
		r.performLinkOperation(ctx, r.delete, result, resp.State, &resp.State, &resp.Diagnostics)
		return
	}

	tflog.Debug(ctx, "about to store private creation result", map[string]any{"encoded result": resultEncoded})
	diag := resp.Private.SetKey(ctx, createResultKey, resultEncoded)
	resp.Diagnostics.Append(diag...)
}

func (r *MgcConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx = tflog.SetField(ctx, rpcField, "read")
	ctx = tflog.SetField(ctx, connectionResourceNameField, r.name)
	r.performLinkOperationFromScratch(ctx, r.read, req.Private.GetKey, resp.Private.SetKey, req.State, &resp.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		// When reading fails, that means that the resource was most likely altered outside of terraform.
		resp.Diagnostics.AddError("reading the resource failed", "was the resource altered outside of terraform?")
		return
	}
	tflog.Info(ctx, "resource read")
}

// Update will most likely never be called, as we always require replace when changed
func (r *MgcConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	ctx = tflog.SetField(ctx, rpcField, "update")
	ctx = tflog.SetField(ctx, connectionResourceNameField, r.name)
	r.performLinkOperationFromScratch(ctx, r.update, req.Private.GetKey, resp.Private.SetKey, tfsdk.State(req.Plan), &resp.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "resource updated")
}

func (r *MgcConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	ctx = tflog.SetField(ctx, rpcField, "delete")
	ctx = tflog.SetField(ctx, connectionResourceNameField, r.name)
	r.performLinkOperationFromScratch(ctx, r.delete, req.Private.GetKey, req.Private.SetKey, req.State, &resp.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "resource deleted")
}

func (r *MgcConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ resource.Resource = (*MgcConnectionResource)(nil)

// END: Resource implementation
