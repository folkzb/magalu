package provider

import (
	"context"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/exp/slices"
	mgcSdk "magalu.cloud/sdk"
)

type tfStateConverter struct {
	ctx      context.Context
	diag     *diag.Diagnostics
	tfSchema *schema.Schema
}

func newTFStateConverter(ctx context.Context, diag *diag.Diagnostics, tfSchema *schema.Schema) tfStateConverter {
	return tfStateConverter{
		ctx:      ctx,
		diag:     diag,
		tfSchema: tfSchema,
	}
}

func (c *tfStateConverter) getEnumType(v *mgcSdk.Schema) string {
	types := []string{}
	for _, v := range v.Enum {
		var t string
		switch v.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			t = "integer"
		case float32, float64:
			t = "number"
		case string:
			t = "string"
		case bool:
			t = "boolean"
		default:
			c.diag.AddError("Unsupported enum value", fmt.Sprintf("unsupported enum value: %+v", v))
			return ""
		}
		if !slices.Contains(types, t) {
			types = append(types, t)
		}
	}
	if len(types) != 1 {
		c.diag.AddError("Multiple types of value in enum", fmt.Sprintf("must provide values of a single type in a enum, got %+v", types))
		return ""
	}

	return types[0]
}

func (c *tfStateConverter) getAttributeType(v *mgcSdk.Schema) string {
	if v.Type == "" {
		if len(v.Enum) != 0 {
			return c.getEnumType(v)
		}

		c.diag.AddError("Unknown value", fmt.Sprintf("[convert] unable to find schema %+v type", v))
	}
	return v.Type
}

func (c *tfStateConverter) toMgcSchemaValue(mgcSchema *mgcSdk.Schema, atinfo *attribute, value tftypes.Value, ignoreUnknown bool, filterUnset bool) (converted any) {
	if mgcSchema == nil {
		c.diag.AddError("Invalid schema", "null schema provided to convert state to go values")
		return nil
	}

	if !value.IsKnown() {
		if !ignoreUnknown {
			c.diag.AddError("Unable to convert unknown value", fmt.Sprintf("[convert] unable to convert since value is unknown: value %+v - schema: %+v", value, mgcSchema))
			return nil
		}
		return nil
	}

	if value.IsNull() {
		if atinfo.isOptional && !atinfo.isComputed {
			// Optional values that aren't computed will never be unknown
			// this means they will be null in the state
			return nil
		} else if !mgcSchema.Nullable && mgcSchema.Type != "null" {
			c.diag.AddError("Unable to convert non nullable value", fmt.Sprintf("[convert] unable to convert since value is null and not nullable by the schema: value %+v - schema: %+v", value, mgcSchema))
			return nil
		}
		return nil
	}

	t := c.getAttributeType(mgcSchema)
	if c.diag.HasError() {
		return nil
	}

	switch t {
	case "string":
		var state string
		err := value.As(&state)
		if err != nil {
			c.diag.AddError("Unable to convert value to string", fmt.Sprintf("[convert] unable to convert value %+v to schema %+v - error: %s", value, mgcSchema, err.Error()))
			return nil
		}
		return state
	case "number":
		var state big.Float
		err := value.As(&state)
		if err != nil {
			c.diag.AddError("Unable to convert value to number", fmt.Sprintf("[convert] unable to convert value %+v to schema %+v - error: %s", value, mgcSchema, err.Error()))
			return nil
		}

		result, accuracy := state.Float64()
		if accuracy != big.Exact {
			c.diag.AddError("Unable to convert value to float", fmt.Sprintf("[convert] value %+v lost accuracy in conversion to %+v", state, result))
			return nil
		}
		return result
	case "integer":
		var state big.Float
		err := value.As(&state)
		if err != nil {
			c.diag.AddError("Unable to convert value to integer", fmt.Sprintf("[convert] unable to convert value %+v to schema %+v - error: %s", value, mgcSchema, err.Error()))
			return nil
		}

		result, accuracy := state.Int64()
		if accuracy != big.Exact {
			c.diag.AddError("Unable to convert value to integer", fmt.Sprintf("[convert] value %+v lost accuracy in conversion to %+v", state, result))
			return nil
		}
		return result
	case "boolean":
		var state bool
		err := value.As(&state)
		if err != nil {
			c.diag.AddError("Unable to convert value to boolean", fmt.Sprintf("[convert] unable to convert value %+v to schema %+v - error: %s", value, mgcSchema, err.Error()))
			return nil
		}
		return state
	case "array":
		state := c.toMgcSchemaArray(mgcSchema, atinfo, value, ignoreUnknown, filterUnset)
		return state
	case "object":
		state := c.toMgcSchemaMap(mgcSchema, atinfo, value, ignoreUnknown, filterUnset)
		return state
	default:
		c.diag.AddError("Unknown value", fmt.Sprintf("[convert] unable to convert value %+v to schema %+v", value, mgcSchema))
		return nil
	}
}

func (c *tfStateConverter) toMgcSchemaArray(mgcSchema *mgcSdk.Schema, atinfo *attribute, value tftypes.Value, ignoreUnknown bool, filterUnset bool) (converted []any) {
	var state []tftypes.Value
	err := value.As(&state)
	if err != nil {
		c.diag.AddError("Unable to convert value to list", fmt.Sprintf("[convert] unable to convert value %+v to schema %+v - error: %s", value, mgcSchema, err.Error()))
		return nil
	}

	// TODO: Handle attribute information in list - it should be mapped to "0" key
	tfinfo := atinfo.attributes["0"]
	converted = make([]any, len(state))
	for i, value := range state {
		lresult := c.toMgcSchemaValue((*mgcSdk.Schema)(mgcSchema.Items.Value), tfinfo, value, ignoreUnknown, filterUnset)
		if c.diag.HasError() {
			c.diag.AddError("Unable to convert array", fmt.Sprintf("unknown value inside array at %v", i))
			return nil
		}
		converted[i] = lresult
	}
	return converted
}

func (c *tfStateConverter) toMgcSchemaMap(mgcSchema *mgcSdk.Schema, atinfo *attribute, value tftypes.Value, ignoreUnknown bool, filterUnset bool) (mgcState map[string]any) {
	var tfState map[string]tftypes.Value
	err := value.As(&tfState)
	if err != nil {
		c.diag.AddError("Unable to convert value to map", fmt.Sprintf("[convert] unable to convert value %+v to schema %+v - error: %s", value, mgcSchema, err.Error()))
		return nil
	}

	mgcState = map[string]any{}
	for attr, ref := range mgcSchema.Properties {
		mgcName := mgcName(attr)
		tfinfo := atinfo.attributes[mgcName]
		if tfinfo == nil {
			c.diag.AddError(
				"Schema attribute missing from attribute information",
				fmt.Sprintf("[convert] schema attribute `%s` doesn't have attribute information", mgcName),
			)
			continue
		}

		tfName := tfinfo.tfName
		v, ok := tfState[string(tfName)]
		if !ok {
			title := "Schema attribute missing from state value"
			msg := fmt.Sprintf("[convert] schema attribute `%s` with info `%+v` missing from state %+v", mgcName, atinfo, tfState)
			if tfinfo.isRequired {
				c.diag.AddError(title, msg)
				return
			}
			tflog.Debug(c.ctx, msg)
			continue
		}

		propValue := c.toMgcSchemaValue((*mgcSdk.Schema)(ref.Value), tfinfo, v, ignoreUnknown, filterUnset)
		if c.diag.HasError() {
			return nil
		}

		// We need to ignore nil values generated from unknown elements that didn't generate errors
		if propValue == nil {
			if tfinfo.isOptional && !tfinfo.isComputed && !filterUnset {
				mgcState[string(mgcName)] = propValue
			} else if ref.Value.Nullable || ref.Value.Type == "null" {
				mgcState[string(mgcName)] = propValue
			}
		} else {
			mgcState[string(mgcName)] = propValue
		}
	}
	return mgcState
}

// Convert MGC schema keys to the corresponding TF state keys.
//
// This is necessary to allow merging an object that has diverging input and output elements.
// For example: "desired_status" and "current_status"
//
// Verify for errors in the converter diagnostics attribute.
func (c *tfStateConverter) mgcKeysToStateKeys(atinfo *attribute, obj map[string]any) {
	if atinfo.mgcSchema == nil {
		c.diag.AddError("Missing schema to conversion", "schema not provided")
		return
	}

	// TODO: Make recursive
	for mgcName, info := range atinfo.attributes {
		value, ok := obj[string(mgcName)]
		if !ok {
			// Ignore non existing values
			continue
		}

		if string(mgcName) != string(info.tfName) {
			obj[string(info.tfName)] = value
			delete(obj, string(mgcName))
		}
	}
}

func (c *tfStateConverter) fromMap(result map[string]any) *tftypes.Value {
	if c.tfSchema == nil {
		// TODO: Error
		return nil
	}

	state := map[string]tftypes.Value{}

	for k, tfAttr := range c.tfSchema.Attributes {
		t := tfAttr.GetType().TerraformType(c.ctx)
		if t.Is(tftypes.List{}) {
			// TODO: Convert list
		} else if t.Is(tftypes.Object{}) {
			// TODO: Convert object
		} else {
			state[k] = tftypes.NewValue(t, result[k])
		}

	}

	finalState := tftypes.NewValue(c.tfSchema.Type().TerraformType(c.ctx), state)
	return &finalState
}
