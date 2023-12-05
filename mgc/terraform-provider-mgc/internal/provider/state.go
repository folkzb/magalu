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

	InputAttributes() mgcAttributes
	OutputAttributes() mgcAttributes
	SplitAttributes() []splitMgcAttribute
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
func readMgcMap(handler tfStateHandler, mgcSchema *mgcSdk.Schema, ctx context.Context, tfState tfsdk.State, diag *diag.Diagnostics) map[string]any {
	result := readMgcInputMap(handler, mgcSchema, ctx, tfState, diag)
	output := readMgcOutputMap(handler, mgcSchema, ctx, tfState, diag)
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
func readMgcInputMap(handler tfStateHandler, mgcSchema *mgcSdk.Schema, ctx context.Context, tfState tfsdk.State, diag *diag.Diagnostics) map[string]any {
	conv := newTFStateConverter(ctx, diag, handler.TFSchema())
	return conv.readMgcMap(mgcSchema, handler.InputAttributes(), tfState)
}

// If the Output Attributes don't have an attribute requested by 'mgcSchema', it will be ignored, but no errors will be diagnosed
func readMgcOutputMap(handler tfStateHandler, mgcSchema *mgcSdk.Schema, ctx context.Context, tfState tfsdk.State, diag *diag.Diagnostics) map[string]any {
	conv := newTFStateConverter(ctx, diag, handler.TFSchema())
	return conv.readMgcMap(mgcSchema, handler.OutputAttributes(), tfState)
}

func applyMgcInputMap(handler tfStateHandler, mgcMap map[string]any, ctx context.Context, tfState *tfsdk.State, diag *diag.Diagnostics) {
	conv := newTFStateConverter(ctx, diag, handler.TFSchema())
	conv.applyMgcMap(mgcMap, handler.InputAttributes(), ctx, tfState, path.Empty())
}

func applyMgcOutputMap(handler tfStateHandler, mgcMap map[string]any, ctx context.Context, tfState *tfsdk.State, diag *diag.Diagnostics) {
	conv := newTFStateConverter(ctx, diag, handler.TFSchema())
	conv.applyMgcMap(mgcMap, handler.OutputAttributes(), ctx, tfState, path.Empty())
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
func castToMap(result core.ResultWithValue, diag *diag.Diagnostics) map[string]any {
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
	diag *diag.Diagnostics,
) {
	tflog.Debug(ctx, fmt.Sprintf("[resource] applying state after for %q", handler.Name()))

	// First, apply the values that the user passed as Parameters to the state (assuming success)
	applyMgcInputMap(handler, result.Source().Parameters, ctx, tfState, diag)

	resultMap := castToMap(result, diag)
	if diag.HasError() {
		return
	}
	// Then, apply the result values of the request that was performed
	applyMgcOutputMap(handler, resultMap, ctx, tfState, diag)
	verifyCurrentDesiredMismatch(handler, result.Source().Parameters, resultMap, diag)

	// Then, try to retrieve the Read executor for the Resource via links ('read' parameter may have been nil)
	read = getReadForApplyStateAfter(ctx, handler, result, read, diag)
	if diag.HasError() {
		return
	}

	// If the original result schema is the same as the resource read, no need for further state appliance
	if mgcSchemaPkg.CheckSimilarJsonSchemas(result.Schema(), read.ResultSchema()) {
		return
	}

	paramsSchema := read.ParametersSchema()
	params := readMgcMap(handler, paramsSchema, ctx, *tfState, diag)
	if err := paramsSchema.VisitJSON(params); err != nil {
		diag.AddError(
			"applying state after failed",
			fmt.Sprintf("unable to read resource due to insufficient state information. %s", err),
		)
		return
	}

	readResult := execute(handler.Name(), ctx, read, params, core.Configs{}, diag)
	if diag.HasError() {
		return
	}

	readResultMap := castToMap(readResult, diag)
	if diag.HasError() {
		return
	}

	applyMgcOutputMap(handler, readResultMap, ctx, tfState, diag)
	verifyCurrentDesiredMismatch(handler, result.Source().Parameters, readResultMap, diag)
}
