package provider

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/exp/maps"
	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
	mgcSdk "magalu.cloud/sdk"
)

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
func readMgcMapSchemaFromTFState(ctx context.Context, attrTree resAttrInfoTree, mgcSchema *mgcSdk.Schema, tfState tfsdk.State) (map[string]any, Diagnostics) {
	diagnostics := Diagnostics{}

	result, d := readMgcObject(ctx, mgcSchema, attrTree.input, tfState)
	if !d.HasError() {
		return result, diagnostics.AppendReturn(d...)
	}
	output, d := readMgcObject(ctx, mgcSchema, attrTree.output, tfState)
	if !d.HasError() {
		return result, diagnostics.AppendReturn(d...)
	}

	maps.Copy(result, output)
	if missingKeys := missingSchemaKeysInMap(result, mgcSchema, ""); len(missingKeys) > 0 {
		diagnostics.AddError(
			"unable to read MgcMap from Input and Output Terraform attributes",
			fmt.Sprintf("Reading Input and Output attributes into MgcMap didn't match requested schema. Map: %#v, Schema: %#v, Missing Keys: %v", result, mgcSchema, missingKeys),
		)
		return nil, diagnostics
	}

	return result, diagnostics
}

func verifyCurrentDesiredMismatch(inputAttr resAttrInfoMap, inputMgcMap map[string]any, outputMgcMap map[string]any) Diagnostics {
	diagnostics := Diagnostics{}

	for _, desired := range inputAttr {
		current := desired.currentCounterpart
		if current == nil {
			continue
		}

		input, ok := inputMgcMap[string(desired.mgcName)]
		if !ok {
			continue
		}

		output, ok := outputMgcMap[string(current.mgcName)]
		if !ok {
			continue
		}

		if !reflect.DeepEqual(input, output) {
			diagnostics.AddWarning(
				"current/desired attribute mismatch",
				fmt.Sprintf(
					"Terraform isn't able to verify the equality between %q (%v) and %q (%v) because their structures are different. Assuming success.",
					current.tfName,
					output,
					desired.tfName,
					input,
				),
			)
		}
	}
	return diagnostics
}

// Does not return error, check for 'diag.HasError' to see if operation was successful
func resultAsMap(result core.ResultWithValue) (map[string]any, Diagnostics) {
	if result == nil {
		return nil, nil
	}
	resultMap, ok := result.Value().(map[string]any)
	if !ok {
		return nil, NewLocalErrorDiagnostics(
			"Operation output mismatch",
			fmt.Sprintf("Unable to convert %v to map.", result),
		)
	}

	return resultMap, nil
}

// If 'result.Source().Executor' has a "read" link, return that. Otherwise, return 'resourceRead'
func getReadForApplyStateAfter(
	ctx context.Context,
	resourceName tfName,
	result core.ResultWithValue,
	resourceRead core.Executor,
) (core.Executor, Diagnostics) {
	readLink, ok := result.Source().Executor.Links()["get-connection"]
	if !ok {
		readLink, ok = result.Source().Executor.Links()["get"]
	}

	if ok {
		var err error
		resourceRead, err = readLink.CreateExecutor(result)
		if err != nil {
			return nil, NewLocalErrorDiagnostics(
				"Read link failed",
				fmt.Sprintf("Unable to create Read link executor for applying new state on resource %q: %s", resourceName, err),
			)
		}
		tflog.Debug(ctx, "[resource] will read using link")
		return resourceRead, nil
	}

	tflog.Debug(ctx, "[resource] will read using resource read call")

	// Should not happen, but check just in case, to avoid potential crashes
	if resourceRead == nil {
		return nil, NewLocalErrorDiagnostics(
			"applying state after failed",
			fmt.Sprintf("operation has no 'read' link and resource has no 'read' call. Reading is impossible for %q'.", resourceName),
		)
	}

	return resourceRead, nil
}

// 'read' parameter may be nil ONLY IF 'result.Source().Executor' already has a Link named "read"
func applyStateAfter(
	ctx context.Context,
	resourceName tfName,
	attrTree resAttrInfoTree,
	result core.ResultWithValue,
	read core.Executor,
	tfState *tfsdk.State,
) (readResult core.ResultWithValue, readResultMap map[string]any, diagnostics Diagnostics) {
	diagnostics = Diagnostics{}
	tflog.Debug(ctx, fmt.Sprintf("[resource] applying state after for %q", resourceName))

	tflog.Debug(ctx, "[resource] applying request parameters in state")
	// First, apply the values that the user passed as Parameters to the state (assuming success)
	d := applyMgcMapToTFState(ctx, result.Source().Parameters, result.Source().Executor.ParametersSchema(), attrTree.input, tfState)
	if diagnostics.AppendCheckError(d...) {
		return nil, nil, d
	}

	resultMap, d := resultAsMap(result)
	if !d.HasError() {
		tflog.Debug(ctx, "[resource] applying request result in state")
		// Then, apply the result values of the request that was performed
		d := applyMgcMapToTFState(ctx, resultMap, result.Schema(), attrTree.output, tfState)
		if diagnostics.AppendCheckError(d...) {
			return nil, nil, diagnostics
		}

		tflog.Debug(ctx, "[resource] checking result state current/desired mismatches")
		d = verifyCurrentDesiredMismatch(attrTree.input, result.Source().Parameters, resultMap)
		if diagnostics.AppendCheckError(d...) {
			return nil, nil, diagnostics
		}
	} else {
		tflog.Debug(ctx, "[resource] request result was not a map, not applying")
	}

	// Then, try to retrieve the Read executor for the Resource via links ('read' parameter may have been nil)
	read, d = getReadForApplyStateAfter(ctx, resourceName, result, read)
	if diagnostics.AppendCheckError(d...) {
		return nil, nil, diagnostics
	}

	paramsSchema := read.ParametersSchema()
	params, d := readMgcMapSchemaFromTFState(ctx, attrTree, paramsSchema, *tfState)
	if diagnostics.AppendCheckError(d...) {
		return nil, nil, diagnostics
	}

	if err := paramsSchema.VisitJSON(params); err != nil {
		return nil, nil, diagnostics.AppendLocalErrorReturn(
			"applying state after failed",
			fmt.Sprintf("unable to read resource due to insufficient state information. %s", err),
		)
	}

	readResult, ed := execute(ctx, resourceName, read, params, core.Configs{})
	if diagnostics.AppendCheckError(ed.DemoteErrorsToWarnings()...) {
		return nil, nil, diagnostics
	}

	readResultMap, d = resultAsMap(readResult)
	if diagnostics.AppendCheckError(d...) {
		return nil, nil, diagnostics
	}

	d = applyMgcMapToTFState(ctx, readResultMap, readResult.Schema(), attrTree.output, tfState)
	if diagnostics.AppendCheckError(d...) {
		return nil, nil, diagnostics
	}

	d = verifyCurrentDesiredMismatch(attrTree.input, result.Source().Parameters, readResultMap)
	diagnostics.Append(d...)

	return readResult, readResultMap, diagnostics
}
