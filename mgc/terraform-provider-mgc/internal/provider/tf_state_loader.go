package provider

import (
	"context"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	mgcSdk "magalu.cloud/sdk"
)

func loadMgcSchemaValue(ctx context.Context, atinfo *resAttrInfo, tfValue tftypes.Value, ignoreUnknown bool, filterUnset bool) (mgcValue any, isKnown bool, d Diagnostics) {
	tflog.Debug(
		ctx,
		"[loader] starting loading from TF state value to mgc value",
		map[string]any{"mgcName": atinfo.mgcName, "tfName": atinfo.tfName, "value": tfValue},
	)

	mgcSchema := atinfo.mgcSchema
	if mgcSchema == nil {
		return nil, false, NewErrorDiagnostics("Invalid schema", "null schema provided to load state to go values")
	}

	if !tfValue.IsKnown() {
		if !ignoreUnknown {
			return nil, false, NewErrorDiagnostics(
				"Unable to load unknown value",
				fmt.Sprintf("[loader] unable to load %q since value is unknown: value %+v - schema: %+v", atinfo.mgcName, tfValue, mgcSchema),
			)
		}
		tflog.Debug(
			ctx,
			"[loader] value is not known, returning nothing",
			map[string]any{"mgcName": atinfo.mgcName, "tfName": atinfo.tfName, "value": tfValue},
		)
		return nil, false, nil
	}

	if tfValue.IsNull() {
		if filterUnset {
			return nil, false, nil
		}

		if atinfo.tfSchema.IsOptional() && !atinfo.tfSchema.IsComputed() {
			tflog.Debug(
				ctx,
				"[loader] value is null in state due to not being specified, returning null as if unknown",
				map[string]any{"mgcName": atinfo.mgcName, "tfName": atinfo.tfName, "value": tfValue},
			)
			// Optional values that aren't computed will never be unknown
			// this means they will be null in the state
			return nil, false, nil
		} else if !mgcSchema.Nullable {
			return nil, true, NewErrorDiagnostics(
				"Unable to load non nullable value",
				fmt.Sprintf("[loader] unable to load %q since value is null and not nullable by the schema: value %+v - schema: %+v", atinfo.mgcName, tfValue, mgcSchema),
			)
		}
		return nil, true, nil
	}

	switch mgcSchema.Type {
	case "string":
		return loadMgcSchemaString(ctx, atinfo, tfValue, ignoreUnknown, filterUnset)
	case "number":
		return loadMgcSchemaNumber(ctx, atinfo, tfValue, ignoreUnknown, filterUnset)
	case "integer":
		return loadMgcSchemaInt(ctx, atinfo, tfValue, ignoreUnknown, filterUnset)
	case "boolean":
		return loadMgcSchemaBool(ctx, atinfo, tfValue, ignoreUnknown, filterUnset)
	case "array":
		return loadMgcSchemaArray(ctx, atinfo, tfValue, ignoreUnknown, filterUnset)
	case "object":
		return loadMgcSchemaMap(ctx, atinfo, tfValue, ignoreUnknown, filterUnset)
	default:
		return nil, false, NewErrorDiagnostics("Unknown value", fmt.Sprintf("[loader] unable to load %q with value %+v to schema %+v", atinfo.mgcName, tfValue, mgcSchema))
	}
}

func loadMgcSchemaString(ctx context.Context, atinfo *resAttrInfo, tfValue tftypes.Value, ignoreUnknown bool, filterUnset bool) (result any, isKnown bool, d Diagnostics) {
	mgcSchema := atinfo.mgcSchema
	tflog.Debug(
		ctx,
		"[loader] will load as string",
		map[string]any{"mgcName": atinfo.mgcName, "tfName": atinfo.tfName, "value": tfValue, "mgcSchema": mgcSchema},
	)

	var state string
	err := tfValue.As(&state)
	if err != nil {
		return nil, true, NewErrorDiagnostics(
			"Unable to load value to string",
			fmt.Sprintf("[loader] unable to load %q with value %+v to schema %+v - error: %s", atinfo.mgcName, tfValue, mgcSchema, err.Error()),
		)
	}
	tflog.Debug(ctx, "[loader] finished loading string", map[string]any{"tfName": atinfo.tfName, "resulting value": state})
	return state, true, nil
}

func loadMgcSchemaNumber(ctx context.Context, atinfo *resAttrInfo, tfValue tftypes.Value, ignoreUnknown bool, filterUnset bool) (result any, isKnown bool, d Diagnostics) {
	mgcSchema := atinfo.mgcSchema
	tflog.Debug(
		ctx,
		"[loader] will load as number",
		map[string]any{"mgcName": atinfo.mgcName, "tfName": atinfo.tfName, "value": tfValue, "mgcSchema": mgcSchema},
	)

	var state big.Float
	err := tfValue.As(&state)
	if err != nil {
		return nil, true, NewErrorDiagnostics(
			"Unable to load value to number",
			fmt.Sprintf("[loader] unable to load %q with value %+v to schema %+v - error: %s", atinfo.mgcName, tfValue, mgcSchema, err.Error()),
		)
	}

	result, accuracy := state.Float64()
	if accuracy != big.Exact {
		return nil, true, NewErrorDiagnostics(
			"Unable to load value to float",
			fmt.Sprintf("[loader] %q with value %+v lost accuracy in conversion to %+v", atinfo.mgcName, state, result),
		)
	}
	tflog.Debug(ctx, "[loader] finished loading number", map[string]any{"tfName": atinfo.tfName, "resulting value": result})
	return result, true, nil
}

func loadMgcSchemaInt(ctx context.Context, atinfo *resAttrInfo, tfValue tftypes.Value, ignoreUnknown bool, filterUnset bool) (result any, isKnown bool, d Diagnostics) {
	mgcSchema := atinfo.mgcSchema
	tflog.Debug(
		ctx,
		"[loader] will load as int",
		map[string]any{"mgcName": atinfo.mgcName, "tfName": atinfo.tfName, "value": tfValue, "mgcSchema": mgcSchema},
	)

	var state big.Float
	err := tfValue.As(&state)
	if err != nil {
		return nil, true, NewErrorDiagnostics(
			"Unable to load value to integer",
			fmt.Sprintf("[loader] unable to load %q with value %+v to schema %+v - error: %s", atinfo.mgcName, tfValue, mgcSchema, err.Error()),
		)
	}

	result, accuracy := state.Int64()
	if accuracy != big.Exact {
		return nil, true, NewErrorDiagnostics(
			"Unable to load value to integer",
			fmt.Sprintf("[loader] %q with value %+v lost accuracy in conversion to %+v", atinfo.mgcName, state, result),
		)
	}
	tflog.Debug(ctx, "[loader] finished loading integer", map[string]any{"tfName": atinfo.tfName, "resulting value": result})
	return result, true, nil
}

func loadMgcSchemaBool(ctx context.Context, atinfo *resAttrInfo, tfValue tftypes.Value, ignoreUnknown bool, filterUnset bool) (result any, isKnown bool, d Diagnostics) {
	mgcSchema := atinfo.mgcSchema
	tflog.Debug(
		ctx,
		"[loader] will load as bool",
		map[string]any{"mgcName": atinfo.mgcName, "tfName": atinfo.tfName, "value": tfValue, "mgcSchema": mgcSchema},
	)

	var state bool
	err := tfValue.As(&state)
	if err != nil {
		return nil, false, NewErrorDiagnostics(
			"Unable to load value to boolean",
			fmt.Sprintf("[loader] unable to load %q with value %+v to schema %+v - error: %s", atinfo.mgcName, tfValue, mgcSchema, err.Error()),
		)
	}
	tflog.Debug(ctx, "[loader] finished loading bool", map[string]any{"tfName": atinfo.tfName, "resulting value": state})
	return state, true, nil
}

func loadMgcSchemaArray(ctx context.Context, atinfo *resAttrInfo, tfValue tftypes.Value, ignoreUnknown bool, filterUnset bool) (mgcArray []any, isKnown bool, diagnostics Diagnostics) {
	diagnostics = Diagnostics{}
	tflog.Debug(
		ctx,
		"[loader] will load as array",
		map[string]any{"mgcName": atinfo.mgcName, "tfName": atinfo.tfName, "value": tfValue},
	)
	mgcSchema := atinfo.mgcSchema

	var tfArray []tftypes.Value
	err := tfValue.As(&tfArray)
	if err != nil {
		return nil, false, diagnostics.AppendErrorReturn(
			"Unable to load value to list",
			fmt.Sprintf("[loader] unable to load %q with value %+v to schema %+v - error: %s", atinfo.mgcName, tfValue, mgcSchema, err.Error()),
		)
	}

	itemAttr := atinfo.childAttributes["0"]
	mgcArray = make([]any, len(tfArray))
	isKnown = true
	for i, tfItem := range tfArray {
		mgcItem, isItemKnown, d := loadMgcSchemaValue(ctx, itemAttr, tfItem, ignoreUnknown, filterUnset)
		if diagnostics.AppendCheckError(d...) {
			diagnostics.AddError("Unable to load array", fmt.Sprintf("unknown value inside %q array at %v", atinfo.mgcName, i))
			return nil, isItemKnown, diagnostics
		}
		if !isItemKnown {
			// TODO: confirm this logic, should we just keep going?
			diagnostics.AddWarning("Unknown list item", fmt.Sprintf("Item %d in %q is unknown: %+v", i, atinfo.mgcName, tfItem))
			isKnown = false
			return mgcArray, isKnown, diagnostics
		}
		mgcArray[i] = mgcItem
	}
	tflog.Debug(ctx, "[loader] finished loading array", map[string]any{"tfName": atinfo.tfName, "resulting value": mgcArray})
	return mgcArray, isKnown, diagnostics
}

// If 'atinfo' doesn't have a property present in 'atinfo.mgcSchema', it will be ignored. This means that the resulting MgcMap may be incomplete and it is up
// to the caller to ensure that all properties of 'atinfo.mgcSchema' were fulfilled in the resulting mgcMap
func loadMgcSchemaMap(ctx context.Context, atinfo *resAttrInfo, tfValue tftypes.Value, ignoreUnknown bool, filterUnset bool) (any, bool, Diagnostics) {
	diagnostics := Diagnostics{}
	tflog.Debug(
		ctx,
		"[loader] will load as map",
		map[string]any{"mgcName": atinfo.mgcName, "tfName": atinfo.tfName, "value": tfValue},
	)
	var tfMap map[string]tftypes.Value
	err := tfValue.As(&tfMap)
	if err != nil {
		return nil, false, diagnostics.AppendErrorReturn(
			"Unable to load value to map",
			fmt.Sprintf("[loader] unable to load %q with value %+v to schema %+v - error: %s", atinfo.mgcName, tfValue, atinfo.mgcSchema, err.Error()),
		)
	}

	mgcMap := map[string]any{}
	isKnown := true

	var xOfKey string
	var xOfPromotedValues []string

	for propName := range atinfo.mgcSchema.Properties {
		propMgcName := mgcName(propName)

		tflog.Debug(
			ctx,
			"[loader] will try to load map property",
			map[string]any{"propMgcName": propMgcName},
		)

		propAttr, ok := atinfo.childAttributes[propMgcName]
		if !ok {
			tflog.Debug(
				ctx,
				"[loader] ignoring non-existent value",
				map[string]any{"mgcName": propMgcName, "value": tfValue},
			)
			continue
		}

		propTFItem, ok := tfMap[string(propAttr.tfName)]
		if !ok {
			if propAttr.tfSchema.IsRequired() {
				return mgcMap, false, diagnostics.AppendErrorReturn(
					"Schema attribute missing from state value",
					fmt.Sprintf("[loader] schema attribute %q with info `%+v` missing from state %+v", propMgcName, atinfo, tfMap),
				)
			}
			continue
		}

		propMgcItem, isItemKnown, d := loadMgcSchemaValue(ctx, propAttr, propTFItem, ignoreUnknown, filterUnset)
		if diagnostics.AppendCheckError(d...) {
			return nil, false, diagnostics
		}

		if !isItemKnown && ignoreUnknown {
			tflog.Debug(
				ctx,
				"[loader] ignoring prop, unknown",
				map[string]any{"propMgcName": propMgcName, "propTFName": propAttr.tfName, "value": propMgcItem},
			)
			continue
		}
		if propMgcItem == nil && filterUnset {
			tflog.Debug(
				ctx,
				"[loader] ignoring prop, value is nil and 'filterUnset' is set to true",
				map[string]any{"propMgcName": propMgcName, "propTFName": propAttr.tfName},
			)
			continue
		}

		if propXOfKey, ok := propAttr.mgcSchema.Extensions[xOfPromotionKey].(string); ok {
			tflog.Debug(
				ctx,
				fmt.Sprintf("[loader] found xOf promotion key: %q", propXOfKey),
				map[string]any{"propMgcName": propMgcName},
			)
			// TODO: Treat every xOf as "OneOf" for now, so fail if attributes from different xOf children were specified
			if xOfKey == "" {
				xOfKey = propXOfKey
				xOfPromotedValues = append(xOfPromotedValues, propName)
			} else if xOfKey != propXOfKey {
				return mgcMap, false, diagnostics.AppendErrorReturn(
					"mutually exclusive attributes specified",
					fmt.Sprintf("attribute %q cannot be specified if attribute(s) %v have already been specified and vice-versa", propName, xOfPromotedValues),
				)
			}
		}

		if isSchemaXOfAlternative(propAttr.mgcSchema) {
			tflog.Debug(
				ctx,
				"[loader] returning value from map as map itself, since it was a promoted xOf schema",
				map[string]any{"mgcName": atinfo.mgcName, "tfName": atinfo.tfName, "value": propMgcItem},
			)
			return propMgcItem, isKnown, diagnostics
		} else {
			tflog.Debug(
				ctx,
				"[loader] loaded map prop",
				map[string]any{"mgcName": atinfo.mgcName, "tfName": atinfo.tfName, "value": propMgcItem},
			)
			mgcMap[string(propMgcName)] = propMgcItem
		}
	}
	tflog.Debug(ctx, "[loader] finished loading map", map[string]any{"tfName": atinfo.tfName, "resulting value": mgcMap})
	return mgcMap, isKnown, diagnostics
}

// Read values from tfValue into a map suitable to MGC
func readMgcMap(ctx context.Context, mgcSchema *mgcSdk.Schema, attributes resAttrInfoMap, tfState tfsdk.State) (map[string]any, Diagnostics) {
	attr := &resAttrInfo{
		tfName:          "inputSchemasInfo",
		mgcName:         "inputSchemasInfo",
		mgcSchema:       mgcSchema,
		childAttributes: attributes,
	}

	diagnostics := Diagnostics{}
	m, _, d := loadMgcSchemaMap(ctx, attr, tfState.Raw, true, true)
	if diagnostics.AppendCheckError(d...) {
		return nil, diagnostics
	}

	return m.(map[string]any), diagnostics
}
