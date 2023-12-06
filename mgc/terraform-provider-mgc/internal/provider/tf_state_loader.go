package provider

import (
	"context"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	mgcSdk "magalu.cloud/sdk"
)

type tfStateLoader struct {
	ctx      context.Context
	diag     *diag.Diagnostics
	tfSchema *schema.Schema
}

func newTFStateLoader(ctx context.Context, diag *diag.Diagnostics, tfSchema *schema.Schema) tfStateLoader {
	return tfStateLoader{
		ctx:      ctx,
		diag:     diag,
		tfSchema: tfSchema,
	}
}

func (c *tfStateLoader) loadMgcSchemaValue(atinfo *resAttrInfo, tfValue tftypes.Value, ignoreUnknown bool, filterUnset bool) (mgcValue any, isKnown bool) {
	tflog.Debug(
		c.ctx,
		"[loader] starting loading from TF state value to mgc value",
		map[string]any{"mgcName": atinfo.mgcName, "tfName": atinfo.tfName, "value": tfValue},
	)
	mgcSchema := atinfo.mgcSchema
	if mgcSchema == nil {
		c.diag.AddError("Invalid schema", "null schema provided to load state to go values")
		return nil, false
	}

	if !tfValue.IsKnown() {
		if !ignoreUnknown {
			c.diag.AddError(
				"Unable to load unknown value",
				fmt.Sprintf("[loader] unable to load %q since value is unknown: value %+v - schema: %+v", atinfo.mgcName, tfValue, mgcSchema),
			)
			return nil, false
		}
		return nil, false
	}

	if tfValue.IsNull() {
		if atinfo.tfSchema.IsOptional() && !atinfo.tfSchema.IsComputed() {
			// Optional values that aren't computed will never be unknown
			// this means they will be null in the state
			return nil, true
		} else if !mgcSchema.Nullable {
			c.diag.AddError(
				"Unable to load non nullable value",
				fmt.Sprintf("[loader] unable to load %q since value is null and not nullable by the schema: value %+v - schema: %+v", atinfo.mgcName, tfValue, mgcSchema),
			)
			return nil, true
		}
		return nil, true
	}

	switch mgcSchema.Type {
	case "string":
		return c.loadMgcSchemaString(atinfo, tfValue, ignoreUnknown, filterUnset)
	case "number":
		return c.loadMgcSchemaNumber(atinfo, tfValue, ignoreUnknown, filterUnset)
	case "integer":
		return c.loadMgcSchemaInt(atinfo, tfValue, ignoreUnknown, filterUnset)
	case "boolean":
		return c.loadMgcSchemaBool(atinfo, tfValue, ignoreUnknown, filterUnset)
	case "array":
		return c.loadMgcSchemaArray(atinfo, tfValue, ignoreUnknown, filterUnset)
	case "object":
		return c.loadMgcSchemaMap(atinfo, tfValue, ignoreUnknown, filterUnset)
	default:
		c.diag.AddError("Unknown value", fmt.Sprintf("[loader] unable to load %q with value %+v to schema %+v", atinfo.mgcName, tfValue, mgcSchema))
		return nil, false
	}
}

func (c *tfStateLoader) loadMgcSchemaString(atinfo *resAttrInfo, tfValue tftypes.Value, ignoreUnknown bool, filterUnset bool) (result any, isKnown bool) {
	mgcSchema := atinfo.mgcSchema
	tflog.Debug(
		c.ctx,
		"[loader] will load as string",
		map[string]any{"mgcName": atinfo.mgcName, "tfName": atinfo.tfName, "value": tfValue, "mgcSchema": mgcSchema},
	)

	var state string
	err := tfValue.As(&state)
	if err != nil {
		c.diag.AddError(
			"Unable to load value to string",
			fmt.Sprintf("[loader] unable to load %q with value %+v to schema %+v - error: %s", atinfo.mgcName, tfValue, mgcSchema, err.Error()),
		)
		return nil, true
	}
	tflog.Debug(c.ctx, "[loader] finished loading string", map[string]any{"tfName": atinfo.tfName, "resulting value": state})
	return state, true
}

func (c *tfStateLoader) loadMgcSchemaNumber(atinfo *resAttrInfo, tfValue tftypes.Value, ignoreUnknown bool, filterUnset bool) (result any, isKnown bool) {
	mgcSchema := atinfo.mgcSchema
	tflog.Debug(
		c.ctx,
		"[loader] will load as number",
		map[string]any{"mgcName": atinfo.mgcName, "tfName": atinfo.tfName, "value": tfValue, "mgcSchema": mgcSchema},
	)

	var state big.Float
	err := tfValue.As(&state)
	if err != nil {
		c.diag.AddError(
			"Unable to load value to number",
			fmt.Sprintf("[loader] unable to load %q with value %+v to schema %+v - error: %s", atinfo.mgcName, tfValue, mgcSchema, err.Error()),
		)
		return nil, true
	}

	result, accuracy := state.Float64()
	if accuracy != big.Exact {
		c.diag.AddError(
			"Unable to load value to float",
			fmt.Sprintf("[loader] %q with value %+v lost accuracy in conversion to %+v", atinfo.mgcName, state, result),
		)
		return nil, true
	}
	tflog.Debug(c.ctx, "[loader] finished loading number", map[string]any{"tfName": atinfo.tfName, "resulting value": result})
	return result, true
}

func (c *tfStateLoader) loadMgcSchemaInt(atinfo *resAttrInfo, tfValue tftypes.Value, ignoreUnknown bool, filterUnset bool) (result any, isKnown bool) {
	mgcSchema := atinfo.mgcSchema
	tflog.Debug(
		c.ctx,
		"[loader] will load as int",
		map[string]any{"mgcName": atinfo.mgcName, "tfName": atinfo.tfName, "value": tfValue, "mgcSchema": mgcSchema},
	)

	var state big.Float
	err := tfValue.As(&state)
	if err != nil {
		c.diag.AddError(
			"Unable to load value to integer",
			fmt.Sprintf("[loader] unable to load %q with value %+v to schema %+v - error: %s", atinfo.mgcName, tfValue, mgcSchema, err.Error()),
		)
		return nil, true
	}

	result, accuracy := state.Int64()
	if accuracy != big.Exact {
		c.diag.AddError(
			"Unable to load value to integer",
			fmt.Sprintf("[loader] %q with value %+v lost accuracy in conversion to %+v", atinfo.mgcName, state, result),
		)
		return nil, true
	}
	tflog.Debug(c.ctx, "[loader] finished loading integer", map[string]any{"tfName": atinfo.tfName, "resulting value": result})
	return result, true
}

func (c *tfStateLoader) loadMgcSchemaBool(atinfo *resAttrInfo, tfValue tftypes.Value, ignoreUnknown bool, filterUnset bool) (result any, isKnown bool) {
	mgcSchema := atinfo.mgcSchema
	tflog.Debug(
		c.ctx,
		"[loader] will load as bool",
		map[string]any{"mgcName": atinfo.mgcName, "tfName": atinfo.tfName, "value": tfValue, "mgcSchema": mgcSchema},
	)

	var state bool
	err := tfValue.As(&state)
	if err != nil {
		c.diag.AddError(
			"Unable to load value to boolean",
			fmt.Sprintf("[loader] unable to load %q with value %+v to schema %+v - error: %s", atinfo.mgcName, tfValue, mgcSchema, err.Error()),
		)
		return nil, false
	}
	tflog.Debug(c.ctx, "[loader] finished loading bool", map[string]any{"tfName": atinfo.tfName, "resulting value": state})
	return state, true
}

func (c *tfStateLoader) loadMgcSchemaArray(atinfo *resAttrInfo, tfValue tftypes.Value, ignoreUnknown bool, filterUnset bool) (mgcArray []any, isKnown bool) {
	tflog.Debug(
		c.ctx,
		"[loader] starting loading from TF state value to mgc array",
		map[string]any{"mgcName": atinfo.mgcName, "tfName": atinfo.tfName, "value": tfValue},
	)
	mgcSchema := atinfo.mgcSchema

	var tfArray []tftypes.Value
	err := tfValue.As(&tfArray)
	if err != nil {
		c.diag.AddError(
			"Unable to load value to list",
			fmt.Sprintf("[loader] unable to load %q with value %+v to schema %+v - error: %s", atinfo.mgcName, tfValue, mgcSchema, err.Error()),
		)
		return nil, false
	}

	itemAttr := atinfo.childAttributes["0"]
	mgcArray = make([]any, len(tfArray))
	isKnown = true
	for i, tfItem := range tfArray {
		mgcItem, isItemKnown := c.loadMgcSchemaValue(itemAttr, tfItem, ignoreUnknown, filterUnset)
		if c.diag.HasError() {
			c.diag.AddError("Unable to load array", fmt.Sprintf("unknown value inside %q array at %v", atinfo.mgcName, i))
			return nil, isItemKnown
		}
		if !isItemKnown {
			// TODO: confirm this logic, should we just keep going?
			c.diag.AddWarning("Unknown list item", fmt.Sprintf("Item %d in %q is unknown: %+v", i, atinfo.mgcName, tfItem))
			isKnown = false
			return
		}
		mgcArray[i] = mgcItem
	}
	tflog.Debug(c.ctx, "[loader] finished loading array", map[string]any{"tfName": atinfo.tfName, "resulting value": mgcArray})
	return
}

// If 'atinfo' doesn't have a property present in 'atinfo.mgcSchema', it will be ignored. This means that the resulting MgcMap may be incomplete and it is up
// to the caller to ensure that all properties of 'atinfo.mgcSchema' were fulfilled in the resulting mgcMap
func (c *tfStateLoader) loadMgcSchemaMap(atinfo *resAttrInfo, tfValue tftypes.Value, ignoreUnknown bool, filterUnset bool) (mgcMap map[string]any, isKnown bool) {
	tflog.Debug(
		c.ctx,
		"[loader] starting loading TF state value to mgc map",
		map[string]any{"mgcName": atinfo.mgcName, "tfName": atinfo.tfName, "value": tfValue},
	)
	mgcSchema := atinfo.mgcSchema
	var tfMap map[string]tftypes.Value
	err := tfValue.As(&tfMap)
	if err != nil {
		c.diag.AddError(
			"Unable to load value to map",
			fmt.Sprintf("[loader] unable to load %q with value %+v to schema %+v - error: %s", atinfo.mgcName, tfValue, mgcSchema, err.Error()),
		)
		return nil, false
	}

	mgcMap = map[string]any{}
	isKnown = true
	for attr := range mgcSchema.Properties {
		mgcName := mgcName(attr)
		itemAttr, ok := atinfo.childAttributes[mgcName]
		if !ok {
			// Ignore non existing values
			continue
		}

		tfName := itemAttr.tfName
		tfItem, ok := tfMap[string(tfName)]
		if !ok {
			title := "Schema attribute missing from state value"
			msg := fmt.Sprintf("[loader] schema attribute %q with info `%+v` missing from state %+v", mgcName, atinfo, tfMap)
			if itemAttr.tfSchema.IsRequired() {
				c.diag.AddError(title, msg)
				return
			}
			tflog.Debug(c.ctx, msg)
			continue
		}

		mgcItem, isItemKnown := c.loadMgcSchemaValue(itemAttr, tfItem, ignoreUnknown, filterUnset)
		if c.diag.HasError() {
			return nil, false
		}

		if !isItemKnown && ignoreUnknown {
			continue
		}
		if mgcItem == nil && filterUnset {
			continue
		}

		mgcMap[string(mgcName)] = mgcItem
	}
	tflog.Debug(c.ctx, "[loader] finished loading map", map[string]any{"tfName": atinfo.tfName, "resulting value": mgcMap})
	return
}

// Read values from tfValue into a map suitable to MGC
func (c *tfStateLoader) readMgcMap(mgcSchema *mgcSdk.Schema, attributes resAttrInfoMap, tfState tfsdk.State) (mgcMap map[string]any) {
	attr := &resAttrInfo{
		tfName:          "inputSchemasInfo",
		mgcName:         "inputSchemasInfo",
		mgcSchema:       mgcSchema,
		childAttributes: attributes,
	}

	m, _ := c.loadMgcSchemaMap(attr, tfState.Raw, true, true)
	return m
}
