package provider

import (
	"context"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"golang.org/x/exp/slices"
	mgcSdk "magalu.cloud/sdk"
)

type converter struct {
	ctx  context.Context
	diag diag.Diagnostics
}

func (c *converter) getAttributeType(v *mgcSdk.Schema) string {
	// TODO: Do we need to handle enum values that are not string?
	if v.Type == "" {
		if len(v.Enum) != 0 {
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

		c.diag.AddError("Unknown value", fmt.Sprintf("[convert] unable to find schema %+v type", v))
	}
	return v.Type
}

func (c *converter) convertTFToValue(schema *mgcSdk.Schema, atinfo map[string]*attribute, value tftypes.Value) (converted any) {
	if schema == nil {
		c.diag.AddError("Invalid schema", "null schema provided to convert state to go values")
		return nil
	}

	if value.IsNull() {
		if !schema.Nullable && schema.Type != "null" {
			c.diag.AddError("Unable to convert non nullable value", fmt.Sprintf("[convert] unable to convert since value is null and not nullable by the schema: value %+v - schema: %+v", value, schema))
			return nil
		}
		return nil
	}

	t := c.getAttributeType(schema)
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
		state := c.convertTFArray(schema, atinfo, value)
		return state
	case "object":
		state := c.convertTFMap(schema, atinfo, value)
		return state
	default:
		c.diag.AddError("Unknown value", fmt.Sprintf("[convert] unable to convert value %+v to schema %+v", value, schema))
		return nil
	}
}

func (c *converter) convertTFArray(schema *mgcSdk.Schema, atinfo map[string]*attribute, value tftypes.Value) (converted []any) {
	var state []tftypes.Value
	err := value.As(&state)
	if err != nil {
		c.diag.AddError("Unable to convert value to list", fmt.Sprintf("[convert] unable to convert value %+v to schema %+v - error: %s", value, schema, err.Error()))
		return nil
	}

	// TODO: Handle attribute information in list - it should be mapped to "0" key
	tfinfo := atinfo["0"].attributes
	converted = make([]any, len(state))
	for i, value := range state {
		lresult := c.convertTFToValue((*mgcSdk.Schema)(schema.Items.Value), tfinfo, value)
		if c.diag.HasError() {
			c.diag.AddError("Unable to convert array", fmt.Sprintf("unknown value inside array at %v", i))
			return nil
		}
		converted[i] = lresult
	}
	return converted
}

func (c *converter) convertTFMap(schema *mgcSdk.Schema, atinfo map[string]*attribute, value tftypes.Value) (converted map[string]any) {
	var state map[string]tftypes.Value
	err := value.As(&state)
	if err != nil {
		c.diag.AddError("Unable to convert value to map", fmt.Sprintf("[convert] unable to convert value %+v to schema %+v - error: %s", value, schema, err.Error()))
		return nil
	}

	converted = map[string]any{}
	for schemaKey, propSchema := range schema.Properties {
		tfinfo := atinfo[schemaKey]
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

		propValue := c.convertTFToValue((*mgcSdk.Schema)(propSchema.Value), tfinfo.attributes, v)
		if c.diag.HasError() {
			return nil
		}

		converted[schemaKey] = propValue
	}
	return converted
}
