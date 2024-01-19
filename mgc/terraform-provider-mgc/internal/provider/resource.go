package provider

import (
	"context"
	"fmt"

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
	resTfName         tfName
	resMgcName        mgcName
	description       string
	create            mgcSdk.Executor
	read              mgcSdk.Executor
	update            mgcSdk.Executor
	delete            mgcSdk.Executor
	inputAttrInfoMap  resAttrInfoMap
	outputAttrInfoMap resAttrInfoMap
	tfschema          *schema.Schema
	propertySetters   map[mgcName]propertySetter
}

func newMgcResource(
	ctx context.Context,
	sdk *mgcSdk.Sdk,
	resTfName tfName,
	resMgcName mgcName,
	description string,
	create, read, update, delete, list mgcSdk.Executor,
) (*MgcResource, error) {
	if create == nil {
		return nil, fmt.Errorf("resource %q misses create", resTfName)
	}
	if delete == nil {
		return nil, fmt.Errorf("resource %q misses delete", resTfName)
	}
	if read == nil {
		if list == nil {
			return nil, fmt.Errorf("resource %q misses read", resTfName)
		}

		readFromList, err := createReadFromList(list, create.ResultSchema(), delete.ParametersSchema())
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("unable to generate 'read' operation from 'list' for %q", resTfName), map[string]any{"error": err})
			return nil, fmt.Errorf("resource %q misses read", resTfName)
		}

		read = readFromList
		tflog.Debug(ctx, fmt.Sprintf("generated 'read' operation based on 'list' for %q", resTfName))
	}
	if update == nil {
		update = core.NoOpExecutor()
	}

	propertySetterContainer, err := collectPropertySetterContainers(read.Links())
	if err != nil {
		return nil, err
	}

	var propertySetters map[mgcName]propertySetter
	for key, container := range propertySetterContainer {
		if propertySetters == nil {
			propertySetters = make(map[mgcName]propertySetter, len(propertySetterContainer))
		}

		switch container.argCount {
		case 0:
			propertySetters[key] = newDefaultPropertySetter(key, container.entries[0].target)
		case 1:
			propertySetters[key] = newEnumPropertySetter(key, container.entries)
		case 2:
			propertySetters[key] = newStrTransitionPropertySetter(key, container.entries)
		default:
			// TODO: Handle more action types?
			continue
		}
	}

	return &MgcResource{
		sdk:             sdk,
		resTfName:       resTfName,
		resMgcName:      resMgcName,
		description:     description,
		create:          create,
		read:            read,
		update:          update,
		delete:          delete,
		propertySetters: propertySetters,
	}, nil
}

func (r *MgcResource) doesPropHaveSetter(name mgcName) bool {
	_, ok := r.propertySetters[name]
	return ok
}

func (r *MgcResource) getCreateParamsModifiers(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	var k = string(mgcName)
	var nameOverride = mgcName.tfNameOverride(r, mgcSchema)

	if nameOverride != "" {
		k = string(nameOverride)
	}

	isRequired := slices.Contains(mgcSchema.Required, k)
	isComputed := !isRequired
	if isComputed {
		readSchema := r.read.ResultSchema().Properties[k]
		if readSchema == nil {
			isComputed = false
		} else {
			// If not required and present in read it can be compute
			isComputed = mgcSchemaPkg.CheckSimilarJsonSchemas((*core.Schema)(readSchema.Value), (*core.Schema)(mgcSchema.Properties[k].Value))
		}
	}

	return attributeModifiers{
		isRequired:                 isRequired,
		isOptional:                 !isRequired,
		isComputed:                 isComputed,
		useStateForUnknown:         false,
		nameOverride:               nameOverride,
		requiresReplaceWhenChanged: r.update.ParametersSchema().Properties[k] == nil && !r.doesPropHaveSetter(mgcName),
		getChildModifiers:          getInputChildModifiers,
	}
}

func (r *MgcResource) getUpdateParamsModifiers(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	var k = string(mgcName)
	var nameOverride = mgcName.tfNameOverride(r, mgcSchema)

	if nameOverride != "" {
		k = string(nameOverride)
	}

	isComputed := r.create.ResultSchema().Properties[k] != nil
	required := slices.Contains(mgcSchema.Required, k)

	return attributeModifiers{
		isRequired:                 required && !isComputed,
		isOptional:                 !required && !isComputed,
		isComputed:                 !required || isComputed,
		useStateForUnknown:         true,
		nameOverride:               nameOverride,
		requiresReplaceWhenChanged: false,
		getChildModifiers:          getInputChildModifiers,
	}
}

func (r *MgcResource) getDeleteParamsModifiers(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	if _, isInRead := r.read.ParametersSchema().Properties[string(mgcName)]; isInRead {
		return r.getUpdateParamsModifiers(ctx, mgcSchema, mgcName)
	}

	if _, isInUpdate := r.update.ParametersSchema().Properties[string(mgcName)]; isInUpdate {
		return r.getUpdateParamsModifiers(ctx, mgcSchema, mgcName)
	}

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
		nameOverride:               mgcName.tfNameOverride(r, mgcSchema),
		requiresReplaceWhenChanged: false,
		getChildModifiers:          getInputChildModifiers,
	}
}

func (r *MgcResource) getResultModifiers(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	isOptional := r.doesPropHaveSetter(mgcName)
	// In the following cases, this Result Attr will have a different structure when compared to its
	// Input counterpart, and this it will be transformed into 'current_<name>'. These 'current_<name>'
	// attributes, with mismatches, should NEVER be optional, the User must never deal with them
	// directly. The Input counterpart will already have been computed, so this can safely be
	// 'optional == false' since the '<name>' version of this attribute will be 'optional == true'
	// (or 'required == true').
	//
	// Ideally we would make this check when generating the attributes and seeing if there are
	// mismatches, when we transform this into a 'current_<name>' attribute, but we can't modify
	// the 'isOptional' value to false in that step since it's after everything has already been
	// created and we only have an interface to deal with the TF Schema :/
	if createProp, ok := r.create.ParametersSchema().Properties[string(mgcName)]; isOptional && ok {
		isOptional = mgcSchemaPkg.CheckSimilarJsonSchemas((*mgcSchemaPkg.Schema)(createProp.Value), mgcSchema)
	} else if updateProp, ok := r.update.ParametersSchema().Properties[string(mgcName)]; isOptional && ok {
		isOptional = mgcSchemaPkg.CheckSimilarJsonSchemas((*mgcSchemaPkg.Schema)(updateProp.Value), mgcSchema)
	}

	return attributeModifiers{
		isRequired:                 false,
		isOptional:                 isOptional,
		isComputed:                 true,
		useStateForUnknown:         false,
		requiresReplaceWhenChanged: false,
		getChildModifiers:          getResultModifiers,
	}
}

func (r *MgcResource) InputAttrInfoMap(ctx context.Context, d *Diagnostics) resAttrInfoMap {
	if r.inputAttrInfoMap == nil {
		r.inputAttrInfoMap = generateResAttrInfoMap(ctx, r.resTfName,
			[]resAttrInfoGenMetadata{
				{r.create.ParametersSchema(), r.getCreateParamsModifiers},
				{r.update.ParametersSchema(), r.getUpdateParamsModifiers},
				{r.delete.ParametersSchema(), r.getDeleteParamsModifiers},
			}, d,
		)
	}
	return r.inputAttrInfoMap
}

func (r *MgcResource) OutputAttrInfoMap(ctx context.Context, d *Diagnostics) resAttrInfoMap {
	if r.outputAttrInfoMap == nil {
		r.outputAttrInfoMap = generateResAttrInfoMap(ctx, r.resTfName,
			[]resAttrInfoGenMetadata{
				{r.create.ResultSchema(), r.getResultModifiers},
				{r.read.ResultSchema(), r.getResultModifiers},
			}, d,
		)
	}
	return r.outputAttrInfoMap
}

func (r *MgcResource) attrTree(ctx context.Context) (tree resAttrInfoTree, d Diagnostics) {
	return resAttrInfoTree{input: r.InputAttrInfoMap(ctx, &d), output: r.OutputAttrInfoMap(ctx, &d)}, d
}

// BEGIN: Resource implemenation

func (r *MgcResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = string(r.resTfName)
}

func (r *MgcResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	if r.tfschema == nil {
		ctx = tflog.SetField(ctx, resourceNameField, r.resTfName)
		attrTree, d := r.attrTree(ctx)
		resp.Diagnostics.Append(d...)
		if d.HasError() {
			return
		}

		tfs := generateTFSchema(ctx, r.resTfName, r.description, attrTree, (*Diagnostics)(&resp.Diagnostics))
		r.tfschema = &tfs
	}
	resp.Schema = *r.tfschema
}

func (r *MgcResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	diagnostics := Diagnostics{}
	defer func() {
		resp.Diagnostics = diag.Diagnostics(diagnostics)
	}()

	attrTree, d := r.attrTree(ctx)
	if diagnostics.AppendCheckError(d...) {
		return
	}

	createOp := newMgcResourceCreate(r.resTfName, attrTree, r.create, r.read, r.propertySetters)
	readOp := newMgcResourceRead(r.resTfName, attrTree, r.read)
	operationRunner := newMgcOperationRunner(r.sdk, createOp, readOp, tfsdk.State(req.Plan), req.Plan, &resp.State)
	diagnostics = operationRunner.Run(ctx)
}

func (r *MgcResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	diagnostics := Diagnostics{}
	defer func() {
		resp.Diagnostics = diag.Diagnostics(diagnostics)
	}()

	attrTree, d := r.attrTree(ctx)
	if diagnostics.AppendCheckError(d...) {
		return
	}

	operation := newMgcResourceRead(r.resTfName, attrTree, r.read)
	runner := newMgcOperationRunner(r.sdk, operation, operation, req.State, tfsdk.Plan(req.State), &resp.State)
	diagnostics = runner.Run(ctx)
}

func (r *MgcResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	diagnostics := Diagnostics{}
	defer func() {
		resp.Diagnostics = diag.Diagnostics(diagnostics)
	}()

	attrTree, d := r.attrTree(ctx)
	if diagnostics.AppendCheckError(d...) {
		return
	}

	operation := newMgcResourceUpdate(r.resTfName, attrTree, r.update, r.read, r.propertySetters)
	readOp := newMgcResourceRead(r.resTfName, attrTree, r.read)
	runner := newMgcOperationRunner(r.sdk, operation, readOp, req.State, req.Plan, &resp.State)
	diagnostics = runner.Run(ctx)
}

func (r *MgcResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	diagnostics := Diagnostics{}
	defer func() {
		resp.Diagnostics = diag.Diagnostics(diagnostics)
	}()

	attrTree, d := r.attrTree(ctx)
	if diagnostics.AppendCheckError(d...) {
		return
	}

	deleteOp := newMgcResourceDelete(r.resTfName, attrTree, r.delete)
	runner := newMgcOperationRunner(r.sdk, deleteOp, nil, req.State, tfsdk.Plan(req.State), &resp.State)
	diagnostics = runner.Run(ctx)
}

func (r *MgcResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx = tflog.SetField(ctx, rpcField, "import-state")
	ctx = tflog.SetField(ctx, resourceNameField, r.resTfName)
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// END: Resource implemenation
