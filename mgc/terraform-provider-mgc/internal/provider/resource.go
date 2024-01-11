package provider

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"slices"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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
			propertySetters[key] = newDefaultPropertySetter(container.entries[0].target)
		case 1:
			propertySetters[key] = newEnumPropertySetter(container.entries)
		case 2:
			propertySetters[key] = newStrTransitionPropertySetter(container.entries)
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

func (r *MgcResource) performOperation(
	ctx context.Context,
	exec core.Executor,
	inState tfsdk.State,
) (core.ResultWithValue, Diagnostics) {
	diagnostics := Diagnostics{}

	attrTree, d := r.attrTree(ctx)
	if diagnostics.AppendCheckError(d...) {
		return nil, d
	}

	configs := getConfigs(ctx, exec.ConfigsSchema())
	params, d := readMgcMapSchemaFromTFState(ctx, attrTree, exec.ParametersSchema(), inState)
	if diagnostics.AppendCheckError(d...) {
		return nil, d
	}

	result, d := execute(ctx, r.resTfName, exec, params, configs)
	if diagnostics.AppendCheckError(d...) {
		return nil, d
	}

	return result, diagnostics
}

func (r *MgcResource) findAttrWithTFName(ctx context.Context, name tfName) (*resAttrInfo, bool, Diagnostics) {
	diagnostics := Diagnostics{}

	attrTree, d := r.attrTree(ctx)
	if diagnostics.AppendCheckError(d...) {
		return nil, false, diagnostics
	}

	for _, inputAttr := range attrTree.input {
		if inputAttr.tfName == name {
			return inputAttr, true, diagnostics
		}
	}

	for _, outputAttr := range attrTree.output {
		if outputAttr.tfName == name {
			return outputAttr, true, diagnostics
		}
	}

	return nil, false, diagnostics
}

func (r *MgcResource) callPostOpPropertySetters(
	ctx context.Context,
	inState tfsdk.State,
	outState *tfsdk.State,
	readResult core.ResultWithValue,
	readResultMap map[string]any,
) Diagnostics {
	var diagnostics = Diagnostics{}
	var curStateAsMap map[string]tftypes.Value

	err := inState.Raw.As(&curStateAsMap)
	if err != nil {
		tflog.Warn(
			ctx,
			fmt.Sprintf("[resource] unable to get current state as map: %v. Won't look for property setters", err),
			map[string]any{"resource": r.resTfName},
		)
		return diagnostics
	}

	tflog.Debug(
		ctx,
		"[resource] will look for property setters to perform",
		map[string]any{"resource": r.resTfName},
	)
	var calledSetterExecs []core.Executor
	for stateValKey, stateVal := range curStateAsMap {
		tflog.Debug(
			ctx,
			fmt.Sprintf("[resource] looking for property setter for %q", stateValKey),
			map[string]any{"resource": r.resTfName},
		)

		var currentVal any
		for k, v := range readResultMap {
			if mgcName(k).asTFName() == tfName(stateValKey) {
				currentVal = v
				break
			}
		}
		if currentVal == nil {
			tflog.Debug(
				ctx,
				"[resource] unable to find matching key in read result",
				map[string]any{"resource": r.resTfName, "read result": readResultMap},
			)
			continue
		}

		attr, ok, d := r.findAttrWithTFName(ctx, tfName(stateValKey))
		if diagnostics.AppendCheckError(d...) || !ok {
			tflog.Debug(
				ctx,
				"[resource] unable to find attribute corresponding to TF prop key",
				map[string]any{"resource": r.resTfName, "propTFName": stateValKey, "inputAttributes": r.InputAttrInfoMap(ctx, &d), "outputAttr": r.OutputAttrInfoMap(ctx, &d)},
			)
			continue
		}

		targetVal, wasSpecified, d := loadMgcSchemaValue(ctx, attr, stateVal, false, false)
		if d.HasError() { // Don't append the error
			tflog.Debug(
				ctx,
				"[resource] unable to convert TF value to MGC value to update prop, ignoring set",
				map[string]any{"resource": r.resTfName, "propTFName": attr.tfName, "tfValue": stateVal, "convErrors": d.Errors()},
			)
			continue
		}

		if !wasSpecified {
			tflog.Debug(
				ctx,
				"[resource] value was not specified, will not call property setter",
				map[string]any{"resource": r.resTfName, "propTFName": attr.tfName, "tfValue": stateVal},
			)
			continue
		}

		if reflect.DeepEqual(currentVal, targetVal) {
			tflog.Debug(
				ctx,
				"[resource] no need, current state matches desired state",
				map[string]any{"resource": r.resTfName},
			)
			continue
		}

		setter, ok := r.propertySetters[attr.mgcName]
		if !ok {
			tflog.Warn(
				ctx,
				"[resource] unable to find property setter to update prop, will result in error",
				map[string]any{"resource": r.resTfName, "propTFName": attr.tfName, "propMGCName": attr.mgcName},
			)
			continue
		}

		setterExec, run, d := r.callPropertySetter(ctx, inState, outState, attr.tfName, setter, calledSetterExecs, currentVal, targetVal, readResult)
		if diagnostics.AppendCheckError(d...) || !run {
			return diagnostics
		}

		if setterExec != nil {
			calledSetterExecs = append(calledSetterExecs, setterExec)
		}
	}
	return diagnostics
}

func (r *MgcResource) callPropertySetter(
	ctx context.Context,
	inState tfsdk.State,
	outState *tfsdk.State,
	propTFName tfName,
	setter propertySetter,
	alreadyCalled []core.Executor,
	currentVal, targetVal core.Value,
	readResult core.ResultWithValue,
) (exec core.Executor, run bool, diagnostics Diagnostics) {
	diagnostics = Diagnostics{}
	tflog.Debug(
		ctx,
		fmt.Sprintf("[resource] will update param %q with property setter", propTFName),
		map[string]any{"resource": r.resTfName, "currentValue": currentVal, "targetValue": targetVal},
	)
	target, err := setter.getTarget(currentVal, targetVal)
	if err != nil {
		return nil, false, diagnostics.AppendErrorReturn(
			"[resource] property setter fail",
			fmt.Sprintf("unable to call property setter to update parameter %q: %v", propTFName, err),
		)
	}

	setterExec, err := target.CreateExecutor(readResult)
	if err != nil {
		return nil, false, diagnostics.AppendErrorReturn(
			"[resource] property setter fail",
			fmt.Sprintf("unable to call property setter to update parameter %q: %v", propTFName, err),
		)
	}

	// Here, we compare each entry manually (check the name, parameter and result equality) to know if
	// the executor has already been called. Unfortunately we can't simply use reflect.DeepEqual because
	// it returns 'false' immediately if the underlying types have function fields which are not nil.
	// https://pkg.go.dev/reflect#DeepEqual
	for _, calledSetterExec := range alreadyCalled {
		if setterExec.Name() != calledSetterExec.Name() {
			continue
		}

		if !mgcSchemaPkg.CheckSimilarJsonSchemas(setterExec.ParametersSchema(), calledSetterExec.ParametersSchema()) {
			continue
		}

		if !mgcSchemaPkg.CheckSimilarJsonSchemas(setterExec.ResultSchema(), calledSetterExec.ResultSchema()) {
			continue
		}

		tflog.Debug(
			ctx,
			fmt.Sprintf("[resource] skipping %q property setter because it has already been called by another prop", propTFName),
			map[string]any{"resource": r.resTfName},
		)
		return nil, true, diagnostics
	}

	propSetResult, d := r.performOperation(ctx, setterExec, inState)
	if d.HasError() {
		tflog.Debug(
			ctx,
			"[resource] property setter fail",
			map[string]any{"resource": r.resTfName, "prop": propTFName, "error": d.Errors()},
		)
		diagnostics.AddWarning(
			fmt.Sprintf("Error while updating property %q", propTFName),
			fmt.Sprintf(
				"Error while updating property %q: %v. If there were previous updates to other properties, they will be saved, but further updates will be aborted",
				d.Errors(),
				propTFName,
			),
		)
		return nil, false, diagnostics
	}

	attrTree, d := r.attrTree(ctx)
	if diagnostics.AppendCheckError(d...) {
		return nil, false, diagnostics
	}

	_, _, d = applyStateAfter(ctx, r.resTfName, attrTree, propSetResult, r.read, outState)
	diagnostics.Append(d...)

	return setterExec, true, diagnostics
}

func (r *MgcResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	ctx = tflog.SetField(ctx, rpcField, "create")
	ctx = tflog.SetField(ctx, resourceNameField, r.resTfName)
	ctx = r.sdk.WrapContext(ctx)

	diagnostics := Diagnostics{}
	defer func() {
		resp.Diagnostics = diag.Diagnostics(diagnostics)
	}()

	createResult, d := r.performOperation(ctx, r.create, tfsdk.State(req.Plan))
	if diagnostics.AppendCheckError(d...) {
		return
	}

	attrTree, d := r.attrTree(ctx)
	if diagnostics.AppendCheckError(d...) {
		return
	}

	readResult, readResultMap, d := applyStateAfter(ctx, r.resTfName, attrTree, createResult, r.read, &resp.State)
	if diagnostics.AppendCheckError(d...) {
		return
	}

	d = r.callPostOpPropertySetters(ctx, tfsdk.State(req.Plan), &resp.State, readResult, readResultMap)
	if diagnostics.AppendCheckError(d...) {
		return
	}

	tflog.Info(ctx, "resource created")
}

func (r *MgcResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx = tflog.SetField(ctx, rpcField, "read")
	ctx = tflog.SetField(ctx, resourceNameField, r.resTfName)
	ctx = r.sdk.WrapContext(ctx)

	diagnostics := Diagnostics{}
	defer func() {
		resp.Diagnostics = diag.Diagnostics(diagnostics)
	}()

	readResult, d := r.performOperation(ctx, r.read, req.State)
	if diagnostics.AppendCheckError(d...) {
		return
	}

	attrTree, d := r.attrTree(ctx)
	if diagnostics.AppendCheckError(d...) {
		return
	}

	_, _, d = applyStateAfter(ctx, r.resTfName, attrTree, readResult, r.read, &resp.State)
	if diagnostics.AppendCheckError(d...) {
		return
	}

	tflog.Info(ctx, "resource read")
}

func (r *MgcResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	ctx = tflog.SetField(ctx, rpcField, "update")
	ctx = tflog.SetField(ctx, resourceNameField, r.resTfName)
	ctx = r.sdk.WrapContext(ctx)

	diagnostics := Diagnostics{}
	defer func() {
		resp.Diagnostics = diag.Diagnostics(diagnostics)
	}()

	attrTree, d := r.attrTree(ctx)
	if diagnostics.AppendCheckError(d...) {
		return
	}

	params := map[string]any{}
	configs := getConfigs(ctx, r.update.ConfigsSchema())
	required := r.update.ParametersSchema().Required

	planned, d := readMgcMapSchemaFromTFState(ctx, attrTree, r.update.ParametersSchema(), tfsdk.State(req.Plan))
	if diagnostics.AppendCheckError(d...) {
		return
	}

	state, d := readMgcMapSchemaFromTFState(ctx, attrTree, r.update.ParametersSchema(), tfsdk.State(req.State))
	if diagnostics.AppendCheckError(d...) {
		return
	}

	idParamCount := 0
	for prop := range r.update.ParametersSchema().Properties {
		if _, ok := planned[prop]; !ok {
			continue
		}

		if _, ok := r.read.ParametersSchema().Properties[prop]; ok || planned[prop] != state[prop] {
			params[prop] = planned[prop]
			if prop == "id" || strings.HasSuffix(prop, "_id") {
				idParamCount += 1
			}
			continue
		}

		if !slices.Contains(required, prop) {
			continue
		}

		tflog.Warn(
			ctx,
			"update request has unchanged parameters values from current state. This may cause an error",
			map[string]any{
				"parameter": prop,
				"value":     planned[prop],
			},
		)

		params[prop] = planned[prop]
		if prop == "id" || strings.HasSuffix(prop, "_id") {
			idParamCount += 1
		}
	}

	var updateResult core.ResultWithValue
	if len(params) <= idParamCount {
		updateResult, d = r.performOperation(ctx, r.read, tfsdk.State(resp.State))
	} else {
		updateResult, d = execute(ctx, r.resTfName, r.update, params, configs)
	}

	if diagnostics.AppendCheckError(d...) {
		return
	}

	readResult, readResultMap, d := applyStateAfter(ctx, r.resTfName, attrTree, updateResult, r.read, &resp.State)
	if diagnostics.AppendCheckError(d...) {
		return
	}

	d = r.callPostOpPropertySetters(ctx, tfsdk.State(req.Plan), &resp.State, readResult, readResultMap)
	if diagnostics.AppendCheckError(d...) {
		return
	}

	tflog.Info(ctx, "resource updated")
}

func (r *MgcResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	ctx = tflog.SetField(ctx, rpcField, "delete")
	ctx = tflog.SetField(ctx, resourceNameField, r.resTfName)
	ctx = r.sdk.WrapContext(ctx)

	diagnostics := Diagnostics{}
	defer func() {
		resp.Diagnostics = diag.Diagnostics(diagnostics)
	}()

	_, d := r.performOperation(ctx, r.delete, req.State)
	if diagnostics.AppendCheckError(d...) {
		return
	}

	tflog.Info(ctx, "resource deleted")
}

func (r *MgcResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx = tflog.SetField(ctx, rpcField, "import-state")
	ctx = tflog.SetField(ctx, resourceNameField, r.resTfName)
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// END: Resource implemenation
