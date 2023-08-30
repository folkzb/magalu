package provider

import (
	"context"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"golang.org/x/exp/slices"
	mgcSdk "magalu.cloud/sdk"
)

type converter struct {
	ctx  context.Context
	diag *diag.Diagnostics
}

func (c *converter) getEnumType(v *mgcSdk.Schema) string {
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

// ---- TF To Value ----
func (c *converter) getAttributeType(v *mgcSdk.Schema) string {
	if v.Type == "" {
		if len(v.Enum) != 0 {
			return c.getEnumType(v)
		}

		c.diag.AddError("Unknown value", fmt.Sprintf("[convert] unable to find schema %+v type", v))
	}
	return v.Type
}

func (c *converter) convertTFToValue(schema *mgcSdk.Schema, atinfo *attribute, value tftypes.Value, ignoreUnknown bool) (converted any) {
	if schema == nil {
		c.diag.AddError("Invalid schema", "null schema provided to convert state to go values")
		return nil
	}

	if !value.IsKnown() {
		if !ignoreUnknown {
			c.diag.AddError("Unable to convert unknown value", fmt.Sprintf("[convert] unable to convert since value is unknown: value %+v - schema: %+v", value, schema))
			return nil
		}
		return nil
	}

	if value.IsNull() {
		if atinfo.isOptional && !atinfo.isComputed {
			// Optional values that aren't computed will never be unknown
			// this means they will be null in the state
			return nil
		} else if !schema.Nullable && schema.Type != "null" {
			c.diag.AddError("Unable to convert non nullable value", fmt.Sprintf("[convert] unable to convert since value is null and not nullable by the schema: value %+v - schema: %+v", value, schema))
			return nil
		}
		return nil
	}

	t := c.getAttributeType(schema)
	if c.diag.HasError() {
		return nil
	}

	switch t {
	case "string":
		var state string
		err := value.As(&state)
		if err != nil {
			c.diag.AddError("Unable to convert value to string", fmt.Sprintf("[convert] unable to convert value %+v to schema %+v - error: %s", value, schema, err.Error()))
			return nil
		}
		return state
	case "number":
		var state big.Float
		err := value.As(&state)
		if err != nil {
			c.diag.AddError("Unable to convert value to number", fmt.Sprintf("[convert] unable to convert value %+v to schema %+v - error: %s", value, schema, err.Error()))
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
			c.diag.AddError("Unable to convert value to integer", fmt.Sprintf("[convert] unable to convert value %+v to schema %+v - error: %s", value, schema, err.Error()))
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
			c.diag.AddError("Unable to convert value to boolean", fmt.Sprintf("[convert] unable to convert value %+v to schema %+v - error: %s", value, schema, err.Error()))
			return nil
		}
		return state
	case "array":
		state := c.convertTFArray(schema, atinfo, value, ignoreUnknown)
		return state
	case "object":
		state := c.convertTFMap(schema, atinfo, value, ignoreUnknown)
		return state
	default:
		c.diag.AddError("Unknown value", fmt.Sprintf("[convert] unable to convert value %+v to schema %+v", value, schema))
		return nil
	}
}

func (c *converter) convertTFArray(schema *mgcSdk.Schema, atinfo *attribute, value tftypes.Value, ignoreUnknown bool) (converted []any) {
	var state []tftypes.Value
	err := value.As(&state)
	if err != nil {
		c.diag.AddError("Unable to convert value to list", fmt.Sprintf("[convert] unable to convert value %+v to schema %+v - error: %s", value, schema, err.Error()))
		return nil
	}

	// TODO: Handle attribute information in list - it should be mapped to "0" key
	tfinfo := atinfo.attributes["0"]
	converted = make([]any, len(state))
	for i, value := range state {
		lresult := c.convertTFToValue((*mgcSdk.Schema)(schema.Items.Value), tfinfo, value, ignoreUnknown)
		if c.diag.HasError() {
			c.diag.AddError("Unable to convert array", fmt.Sprintf("unknown value inside array at %v", i))
			return nil
		}
		converted[i] = lresult
	}
	return converted
}

func (c *converter) convertTFMap(schema *mgcSdk.Schema, atinfo *attribute, value tftypes.Value, ignoreUnknown bool) (converted map[string]any) {
	var state map[string]tftypes.Value
	err := value.As(&state)
	if err != nil {
		c.diag.AddError("Unable to convert value to map", fmt.Sprintf("[convert] unable to convert value %+v to schema %+v - error: %s", value, schema, err.Error()))
		return nil
	}

	converted = map[string]any{}
	for schemaKey, propSchema := range schema.Properties {
		tfinfo := atinfo.attributes[schemaKey]
		if tfinfo == nil {
			c.diag.AddError(
				"Schema attribute missing from attribute information",
				fmt.Sprintf("[convert] schema attribute `%s` doesn't have attribute information", schemaKey),
			)
			continue
		}

		tfkey := tfinfo.name
		v, ok := state[tfkey]
		if !ok {
			title := "Schema attribute missing from state value"
			msg := fmt.Sprintf("[convert] schema attribute `%s` with info `%+v` missing from state %+v", schemaKey, atinfo, state)
			if tfinfo.isRequired {
				c.diag.AddError(title, msg)
				return
			}

			continue
		}

		propValue := c.convertTFToValue((*mgcSdk.Schema)(propSchema.Value), tfinfo, v, ignoreUnknown)
		if c.diag.HasError() {
			return nil
		}

		// We need to ignore nil values generated from unknown elements that didn't generate errors
		if propValue == nil {
			if propSchema.Value.Nullable || propSchema.Value.Type == "null" {
				converted[schemaKey] = propValue
			}
		} else {
			converted[schemaKey] = propValue
		}
	}
	return converted
}

// ---- END: TF To Value ----

// ---- Value To TF ----

func (c *converter) convertValueToTF(schema *mgcSdk.Schema, result any, p path.Path) *tftypes.Value {
	if schema == nil {
		c.diag.AddError("Missing schema to conversion", "schema not provided")
		return nil
	}

	t := c.getAttributeType(schema)
	if c.diag.HasError() {
		return nil
	}

	var v tftypes.Value = tftypes.Value{}
	switch t {
	case "string":
		if result == nil {
			v = tftypes.NewValue(tftypes.String, nil)
		} else {
			v = tftypes.NewValue(tftypes.String, result.(string))
		}
	case "number", "integer":
		if result == nil {
			v = tftypes.NewValue(tftypes.Number, nil)
		} else {
			v = tftypes.NewValue(tftypes.Number, result.(float64))
		}
	case "boolean":
		if result == nil {
			v = tftypes.NewValue(tftypes.Bool, nil)
		} else {
			v = tftypes.NewValue(tftypes.Bool, result.(bool))
		}
	case "array":
		// TODO: not ignore array attributes
		return nil
		// Implementation
		// list := []tftypes.Value{}
		// elemSchema := (*mgcSdk.Schema)(schema.Items.Value)
		// for i, v := range result.([]any) {
		// 	convElem := c.convertValueToTF(elemSchema, v, p.AtListIndex(i))
		// 	if c.diag.HasError() {
		// 		return nil
		// 	}
		// 	list = append(list, *convElem)
		// }

		// v = tftypes.NewValue(
		// 	tftypes.List{
		// 		ElementType: list[0].Type(),
		// 	},
		// 	list,
		// )
	case "object":
		// TODO: not ignore nested object attributes
		if !p.Equal(path.Empty()) {
			return nil
		}

		m := map[string]tftypes.Value{}
		mt := map[string]tftypes.Type{}
		for key, value := range result.(map[string]any) { // TODO: Mudar para properties
			mvalue := schema.Properties[key]
			if mvalue == nil {
				c.diag.AddWarning("Schema attribute missing", fmt.Sprintf("[convert] schema attribute `%s` missing from state %+v", key, result))
				continue
			}

			tfpath := p.AtName(key)
			attrValue := c.convertValueToTF((*mgcSdk.Schema)(mvalue.Value), value, tfpath)
			if attrValue != nil {
				m[key] = *attrValue
				mt[key] = (*attrValue).Type()
			}
		}

		v = tftypes.NewValue(tftypes.Object{
			AttributeTypes: mt,
		}, m)
	default:
		c.diag.AddError("Unknown value", fmt.Sprintf("[convert] unable to convert value %+v to schema %+v", result, schema))
		return nil
	}

	return &v
}

// ---- END: Value To TF ----
