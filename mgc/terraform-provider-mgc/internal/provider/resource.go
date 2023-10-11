package provider

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/exp/slices"
	"magalu.cloud/core"
	mgcSdk "magalu.cloud/sdk"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &MgcResource{}
var _ resource.ResourceWithImportState = &MgcResource{}

type splitMgcAttribute struct {
	current *attribute
	desired *attribute
}

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
	splitAttr  []splitMgcAttribute
	tfschema   *schema.Schema
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
			isComputed = checkSimilarJsonSchemas((*core.Schema)(readSchema.Value), (*core.Schema)(mgcSchema.Properties[string(mgcName)].Value))
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
	isCreated := r.create.ResultSchema().Properties[k] != nil
	required := slices.Contains(mgcSchema.Required, k)

	return attributeModifiers{
		isRequired:                 required && !isCreated,
		isOptional:                 !required && !isCreated,
		isComputed:                 !required || isCreated,
		useStateForUnknown:         true,
		requiresReplaceWhenChanged: false,
		getChildModifiers:          getInputChildModifiers,
	}
}

func (r *MgcResource) getDeleteParamsModifiers(ctx context.Context, mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	// For now we consider all delete params as optionals, we need to think a way for the user to define
	// required delete params
	return attributeModifiers{
		isRequired:                 false,
		isOptional:                 true,
		isComputed:                 false,
		useStateForUnknown:         true,
		requiresReplaceWhenChanged: false,
		getChildModifiers:          getInputChildModifiers,
	}
}

func (r *MgcResource) ReadInputAttributes(ctx context.Context) diag.Diagnostics {
	d := diag.Diagnostics{}
	if len(r.inputAttr) != 0 {
		return d
	}
	tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: reading input attributes", r.name))

	input := mgcAttributes{}
	err := addMgcSchemaAttributes(
		input,
		r.create.ParametersSchema(),
		r.getCreateParamsModifiers,
		r.name,
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
		r.name,
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
		r.name,
		ctx,
	)
	if err != nil {
		d.AddError("could not create TF input attributes", err.Error())
		return d
	}

	r.inputAttr = input
	return d
}

func (r *MgcResource) ReadOutputAttributes(ctx context.Context) diag.Diagnostics {
	d := diag.Diagnostics{}
	if len(r.outputAttr) != 0 {
		return d
	}
	tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: reading output attributes", r.name))

	output := mgcAttributes{}
	err := addMgcSchemaAttributes(
		output,
		r.create.ResultSchema(),
		getResultModifiers,
		r.name,
		ctx,
	)
	if err != nil {
		d.AddError("could not create TF output attributes", err.Error())
		return d
	}
	err = addMgcSchemaAttributes(
		output,
		r.read.ResultSchema(),
		getResultModifiers,
		r.name,
		ctx,
	)
	if err != nil {
		d.AddError("could not create TF output attributes", err.Error())
		return d
	}

	r.outputAttr = output
	return d
}

func (r *MgcResource) InputAttributes() mgcAttributes {
	return r.inputAttr
}

func (r *MgcResource) OutputAttributes() mgcAttributes {
	return r.outputAttr
}

func (r *MgcResource) AppendSplitAttribute(split splitMgcAttribute) {
	if r.splitAttr == nil {
		r.splitAttr = []splitMgcAttribute{}
	}
	r.splitAttr = append(r.splitAttr, split)
}

var _ tfSchemaHandler = (*MgcResource)(nil)

// END: tfSchemaHandler implementation

// BEGIN: Resource implementation

func (r *MgcResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.name
}

func (r *MgcResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// TODO: Handle nullable values
	tflog.Debug(ctx, fmt.Sprintf("[resource] generating schema for `%s`", r.name))

	if r.tfschema == nil {
		tfs, d := generateTFSchema(r, ctx)
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

func (r *MgcResource) verifyCurrentDesiredMismatch(inputMgcMap map[string]any, outputMgcMap map[string]any, diag *diag.Diagnostics) {
	for _, splitAttr := range r.splitAttr {
		input, ok := inputMgcMap[string(splitAttr.desired.mgcName)]
		if !ok {
			continue
		}

		output, ok := outputMgcMap[string(splitAttr.current.mgcName)]
		if !ok {
			continue
		}

		if !reflect.DeepEqual(input, output) {
			diag.AddWarning(
				"current/desired attribute mismatch",
				fmt.Sprintf(
					"Terraform isn't able to verify the equality between %q (%v) and %q (%v) because their structures are different. Assuming success.",
					splitAttr.current.tfName,
					output,
					splitAttr.desired.tfName,
					input,
				),
			)
		}
	}
}

// Does not return error, check for 'diag.HasError' to see if operation was successful
func castToMap(result core.ResultWithValue, diag *diag.Diagnostics) map[string]any {
	resultMap, ok := result.Value().(map[string]any)
	if !ok {
		diag.AddError(
			"Operation output mismatch",
			fmt.Sprintf("Unable to convert %v to map.", result),
		)
	}
	return resultMap
}

// Does not return error, check for 'diag.HasError' to see if operation was successful
func (r *MgcResource) execute(
	ctx context.Context,
	exec core.Executor,
	params core.Parameters,
	configs core.Configs,
	diag *diag.Diagnostics,
) core.ResultWithValue {
	var result core.Result
	var err error

	tflog.Debug(ctx, fmt.Sprintf("[resource] will %s new %s resource - request info with params: %#v and configs: %#v", exec.Name(), r.name, params, configs))
	if tExec, ok := core.ExecutorAs[core.TerminatorExecutor](exec); ok {
		tflog.Debug(ctx, "[resource] running as TerminatorExecutor")
		result, err = tExec.ExecuteUntilTermination(ctx, params, configs)
	} else {
		tflog.Debug(ctx, "[resource] running as Executor")
		result, err = exec.Execute(ctx, params, configs)
	}
	if err != nil {
		diag.AddError(
			fmt.Sprintf("Unable to %s resource", exec.Name()),
			fmt.Sprintf("Service returned with error: %v", err),
		)
		return nil
	}

	resultWithValue, ok := core.ResultAs[core.ResultWithValue](result)
	if !ok {
		diag.AddError(
			"Operation output mismatch",
			fmt.Sprintf("result has no value %#v", result),
		)
		return nil
	}

	/* TODO:
	if err := validateResult(diag, result); err != nil {
		return
	}
	*/
	_ = validateResult(diag, resultWithValue) // just ignore errors for now

	return resultWithValue
}

func (r *MgcResource) applyStateAfter(
	result core.ResultWithValue,
	ctx context.Context,
	tfState *tfsdk.State,
	diag *diag.Diagnostics,
) {
	var resultMap map[string]any
	resultSchema := result.Schema()

	if checkSimilarJsonSchemas(resultSchema, r.read.ResultSchema()) {
		if resultMap = castToMap(result, diag); diag.HasError() {
			return
		}
	} else {
		readLink, ok := result.Source().Executor.Links()["read"]
		if !ok {
			diag.AddError("Read link failed", fmt.Sprintf("Unable to resolve Read link for applying new state on resource '%s'", r.name))
			return
		}

		additionalParametersSchema := readLink.AdditionalParametersSchema()
		if len(additionalParametersSchema.Required) > 0 {
			diag.AddError("Read link failed", fmt.Sprintf("Unable to resolve parameters on Read link for applying new state on resource '%s'", r.name))
			return
		}

		exec, err := readLink.CreateExecutor(result)
		if err != nil {
			diag.AddError("Read link failed", fmt.Sprintf("Unable to create Read link executor for applying new state on resource '%s': %s", r.name, err))
			return
		}

		result := r.execute(ctx, exec, core.Parameters{}, core.Configs{}, diag)
		if diag.HasError() {
			return
		}

		if resultMap = castToMap(result, diag); diag.HasError() {
			return
		}
	}

	// We must apply the input parameters in the state, considering that the request went successfully.
	// BE CAREFUL: Don't apply Plan.Raw values into the State they might be Unknown! State only handles Known/Null values.
	// Also, this must come BEFORE applying the result to the state, as that should override these values when valid.
	r.applyMgcInputMap(result.Source().Parameters, ctx, tfState, diag)

	r.applyMgcOutputMap(resultMap, ctx, tfState, diag)
	r.verifyCurrentDesiredMismatch(result.Source().Parameters, resultMap, diag)
}

func getConfigs(schema *core.Schema) core.Configs {
	result := core.Configs{}
	for propName, propRef := range schema.Properties {
		prop := (*core.Schema)(propRef.Value)
		if prop.Default != nil {
			result[propName] = prop.Default
		}
	}
	return result
}

func (r *MgcResource) performOperation(ctx context.Context, exec core.Executor, inState tfsdk.State, outState *tfsdk.State, diag *diag.Diagnostics) {
	ctx = r.sdk.WrapContext(ctx)

	configs := getConfigs(exec.ConfigsSchema())
	params := r.readMgcMap(exec.ParametersSchema(), ctx, inState, diag)
	if diag.HasError() {
		return
	}

	result := r.execute(ctx, exec, params, configs, diag)
	if diag.HasError() {
		return
	}

	r.applyStateAfter(result, ctx, outState, diag)
}

func (r *MgcResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.performOperation(ctx, r.create, tfsdk.State(req.Plan), &resp.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, fmt.Sprintf("[resource] created a %q resource", r.name))
}

func (r *MgcResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.performOperation(ctx, r.read, req.State, &resp.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, fmt.Sprintf("[resource] read a %q resource", r.name))
}

func (r *MgcResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.performOperation(ctx, r.update, tfsdk.State(req.Plan), &resp.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, fmt.Sprintf("[resource] updated a %q resource", r.name))
}

func (r *MgcResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.performOperation(ctx, r.delete, req.State, &resp.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, fmt.Sprintf("[resource] deleted a %q resource", r.name))
}

func (r *MgcResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// END: Resource implemenation

func validateResult(d *diag.Diagnostics, result core.ResultWithValue) error {
	err := result.ValidateSchema()
	if err != nil {
		d.AddWarning(
			"Operation output mismatch",
			fmt.Sprintf("Result has invalid structure: %v", err),
		)
	}
	return err
}
