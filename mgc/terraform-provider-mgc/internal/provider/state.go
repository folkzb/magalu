package provider

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/exp/maps"
	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
	mgcSdk "magalu.cloud/sdk"
)

type tfStateHandler interface {
	Name() string

	TFSchema() *schema.Schema
	ReadResultSchema() *mgcSdk.Schema

	InputAttrInfoMap(ctx context.Context, d *diag.Diagnostics) resAttrInfoMap
	OutputAttrInfoMap(ctx context.Context, d *diag.Diagnostics) resAttrInfoMap
	SplitAttributes() []splitResAttribute
}

func missingSchemaKeysInMap(m map[string]core.Value, s *mgcSchemaPkg.Schema, prefix string) (missing []string) {
	for propName, propSchemaRef := range s.Properties {
		var key string
		if prefix == "" {
			key = propName
		} else {
			key = prefix + "." + propName
		}

		prop, inMap := m[propName]
		if !inMap && slices.Contains(s.Required, propName) {
			missing = append(missing, key)
			continue
		}

		propSchema := propSchemaRef.Value

		if subMap, ok := prop.(map[string]any); ok && len(propSchema.Properties) > 0 {
			missing = append(missing, missingSchemaKeysInMap(subMap, (*mgcSchemaPkg.Schema)(propSchema), key)...)
		}
	}
	return
}

// Try to read attributes from both Input and Output to fill a map that matches the 'mgcSchema'. If any values are missing in the end, errors are diagnosed
func readMgcMapSchemaFromTFState(handler tfStateHandler, mgcSchema *mgcSdk.Schema, ctx context.Context, tfState tfsdk.State, diag *diag.Diagnostics) map[string]any {
	result := readMgcInputMapSchemaFromTFState(handler, mgcSchema, ctx, tfState, diag)
	output := readMgcOutputMapSchemaFromTFState(handler, mgcSchema, ctx, tfState, diag)
	maps.Copy(result, output)

	if missingKeys := missingSchemaKeysInMap(result, mgcSchema, ""); len(missingKeys) > 0 {
		diag.AddError(
			"unable to read MgcMap from Input and Output Terraform attributes",
			fmt.Sprintf("Reading Input and Output attributes into MgcMap didn't match requested schema. Map: %#v, Schema: %#v, Missing Keys: %v", result, mgcSchema, missingKeys),
		)
	}

	return result
}

// If the Input Attributes don't have an attribute requested by 'mgcSchema', it will be ignored, but no errors will be diagnosed
func readMgcInputMapSchemaFromTFState(handler tfStateHandler, mgcSchema *mgcSdk.Schema, ctx context.Context, tfState tfsdk.State, diag *diag.Diagnostics) map[string]any {
	loader := newTFStateLoader(ctx, diag)
	return loader.readMgcMap(mgcSchema, handler.InputAttrInfoMap(ctx, diag), tfState)
}

// If the Output Attributes don't have an attribute requested by 'mgcSchema', it will be ignored, but no errors will be diagnosed
func readMgcOutputMapSchemaFromTFState(handler tfStateHandler, mgcSchema *mgcSdk.Schema, ctx context.Context, tfState tfsdk.State, diag *diag.Diagnostics) map[string]any {
	loader := newTFStateLoader(ctx, diag)
	return loader.readMgcMap(mgcSchema, handler.OutputAttrInfoMap(ctx, diag), tfState)
}

func applyMgcInputMapToTFState(handler tfStateHandler, mgcMap map[string]any, ctx context.Context, tfState *tfsdk.State, diag *diag.Diagnostics) {
	applier := newTFStateApplier(ctx, diag, handler.TFSchema())
	applier.applyMgcMap(mgcMap, handler.InputAttrInfoMap(ctx, diag), ctx, tfState, path.Empty())
}

func applyMgcOutputMapToTFState(handler tfStateHandler, mgcMap map[string]any, ctx context.Context, tfState *tfsdk.State, diag *diag.Diagnostics) {
	applier := newTFStateApplier(ctx, diag, handler.TFSchema())
	applier.applyMgcMap(mgcMap, handler.OutputAttrInfoMap(ctx, diag), ctx, tfState, path.Empty())
}

func verifyCurrentDesiredMismatch(handler tfStateHandler, inputMgcMap map[string]any, outputMgcMap map[string]any, diag *diag.Diagnostics) {
	for _, splitAttr := range handler.SplitAttributes() {
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
func resultAsMap(result core.ResultWithValue, diag *diag.Diagnostics) map[string]any {
	if result == nil {
		return map[string]any{}
	}
	resultMap, ok := result.Value().(map[string]any)
	if !ok {
		diag.AddError(
			"Operation output mismatch",
			fmt.Sprintf("Unable to convert %v to map.", result),
		)
	}
	return resultMap
}

// If 'result.Source().Executor' has a "read" link, return that. Otherwise, return 'resourceRead'
func getReadForApplyStateAfter(
	ctx context.Context,
	handler tfStateHandler,
	result core.ResultWithValue,
	resourceRead core.Executor,
	diag *diag.Diagnostics,
) core.Executor {
	if readLink, ok := result.Source().Executor.Links()["read"]; ok {
		var err error
		resourceRead, err = readLink.CreateExecutor(result)
		if err != nil {
			diag.AddError(
				"Read link failed",
				fmt.Sprintf("Unable to create Read link executor for applying new state on resource %q: %s", handler.Name(), err),
			)
			return nil
		}
		tflog.Debug(ctx, "[resource] will read using link")
		return resourceRead
	}

	tflog.Debug(ctx, "[resource] will read using resource read call")

	// Should not happen, but check just in case, to avoid potential crashes
	if resourceRead == nil {
		diag.AddError(
			"applying state after failed",
			fmt.Sprintf("operation has no 'read' link and resource has no 'read' call. Reading is impossible for %q'.", handler.Name()),
		)
		return nil
	}

	return resourceRead
}

// 'read' parameter may be nil ONLY IF 'result.Source().Executor' already has a Link named "read"
func applyStateAfter(
	handler tfStateHandler,
	result core.ResultWithValue,
	read core.Executor,
	ctx context.Context,
	tfState *tfsdk.State,
	d *diag.Diagnostics,
) (readResult core.ResultWithValue, readResultMap map[string]any) {
	tflog.Debug(ctx, fmt.Sprintf("[resource] applying state after for %q", handler.Name()))

	// First, apply the values that the user passed as Parameters to the state (assuming success)
	applyMgcInputMapToTFState(handler, result.Source().Parameters, ctx, tfState, d)

	resultMapConvDiag := diag.Diagnostics{}
	resultMap := resultAsMap(result, &resultMapConvDiag)
	if !resultMapConvDiag.HasError() {
		// Then, apply the result values of the request that was performed
		applyMgcOutputMapToTFState(handler, resultMap, ctx, tfState, d)
		verifyCurrentDesiredMismatch(handler, result.Source().Parameters, resultMap, d)
	}

	// Then, try to retrieve the Read executor for the Resource via links ('read' parameter may have been nil)
	read = getReadForApplyStateAfter(ctx, handler, result, read, d)
	if d.HasError() {
		return nil, nil
	}

	paramsSchema := read.ParametersSchema()
	params := readMgcMapSchemaFromTFState(handler, paramsSchema, ctx, *tfState, d)
	if err := paramsSchema.VisitJSON(params); err != nil {
		d.AddError(
			"applying state after failed",
			fmt.Sprintf("unable to read resource due to insufficient state information. %s", err),
		)
		return nil, nil
	}

	readResult = execute(tfName(handler.Name()), ctx, read, params, core.Configs{}, d)
	if d.HasError() {
		return nil, nil
	}

	readResultMap = resultAsMap(readResult, d)
	if d.HasError() {
		return nil, nil
	}

	applyMgcOutputMapToTFState(handler, readResultMap, ctx, tfState, d)
	verifyCurrentDesiredMismatch(handler, result.Source().Parameters, readResultMap, d)

	return readResult, readResultMap
}
