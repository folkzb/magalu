package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/numberplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stoewer/go-strcase"
	"magalu.cloud/core"
	mgcSdk "magalu.cloud/sdk"
)

func getRequiredOperationAttrs(ctx context.Context, r string, s *mgcSdk.Schema, attrValues *map[string]*mgcSdk.Schema) *map[string]bool {
	attrReqMap := map[string]bool{}

	// Check required values
	for _, rqv := range s.Required {
		rqvEsc := kebabToSnakeCase(rqv)
		attrReqMap[rqvEsc] = true
	}

	// Check for optional attributes
	for k, ref := range s.Properties {
		kEsc := kebabToSnakeCase(k)
		if attrValues != nil {
			(*attrValues)[k] = (*mgcSdk.Schema)(ref.Value)
		}
		tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: attribute `%s` schema value added to create schema", r, k))

		if ok := attrReqMap[kEsc]; !ok {
			attrReqMap[kEsc] = false
		}
	}

	return &attrReqMap
}

func getOperationAttrs(ctx context.Context, r string, s *mgcSdk.Schema, attrValues *map[string]*mgcSdk.Schema) *map[string]struct{} {
	attrMap := map[string]struct{}{}

	for k, ref := range s.Properties {
		kEsc := kebabToSnakeCase(k)
		if attrValues != nil {
			(*attrValues)[k] = (*mgcSdk.Schema)(ref.Value)
		}
		tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: attribute `%s` schema value added to create result schema", r, k))
		attrMap[kEsc] = struct{}{}
	}
	return &attrMap
}

func getTerraformAttributeType(v *mgcSdk.Schema) attr.Type {
	// TODO: Do we need to handle enum values that are not string?
	if v.Type == "" && len(v.Enum) != 0 {
		return types.ListType{
			ElemType: types.StringType,
		}
	}

	switch t := v.Type; t {
	case "string":
		return types.StringType
	case "number":
		return types.Float64Type
	case "integer":
		return types.Int64Type
	case "boolean":
		return types.BoolType
	case "array":
		return types.ListType{
			ElemType: getTerraformAttributeType((*core.Schema)(v.Items.Value)),
		}
	case "object":
		// TODO: How to handle object?
		return nil
	default:
		// TODO: This should never happen
		return nil
	}
}

func sdkToTerraformAttribute(ctx context.Context, values attrValues, c attrConstraints, diag diag.Diagnostics) schema.Attribute {
	var value *mgcSdk.Schema = nil
	if values.create != nil {
		value = values.create
	} else if values.createResult != nil {
		value = values.createResult
	} else if values.update != nil {
		value = values.update
	} else {
		value = values.readResult
	}
	if value == nil {
		tflog.Debug(ctx, "[resource] attribute not found in any schema")
		return nil
	}

	switch t := getTerraformAttributeType(value); t {
	case types.StringType:
		// I wanted to use an interface to define the modifiers regardless of the attr type
		// but couldn't find the interface, it seems everything is redefined for each type
		// https://github.com/hashicorp/terraform-plugin-framework/blob/main/internal/fwschema/fwxschema/attribute_plan_modification.go
		mod := []planmodifier.String{}
		if c.useStateForUnknown {
			mod = append(mod, stringplanmodifier.UseStateForUnknown())
		}
		if c.requiresReplaceWhenChanged {
			mod = append(mod, stringplanmodifier.RequiresReplace())
		}
		return schema.StringAttribute{
			Description:   value.Description,
			Required:      c.isRequired,
			Optional:      c.isOptional,
			Computed:      c.isComputed,
			PlanModifiers: mod,
		}
	case types.NumberType:
		mod := []planmodifier.Number{}
		if c.useStateForUnknown {
			mod = append(mod, numberplanmodifier.UseStateForUnknown())
		}
		if c.requiresReplaceWhenChanged {
			mod = append(mod, numberplanmodifier.RequiresReplace())
		}
		return schema.NumberAttribute{
			Description:   value.Description,
			Required:      c.isRequired,
			Optional:      c.isOptional,
			Computed:      c.isComputed,
			PlanModifiers: mod,
		}
	case types.Int64Type:
		mod := []planmodifier.Int64{}
		if c.useStateForUnknown {
			mod = append(mod, int64planmodifier.UseStateForUnknown())
		}
		if c.requiresReplaceWhenChanged {
			mod = append(mod, int64planmodifier.RequiresReplace())
		}
		return schema.Int64Attribute{
			Description:   value.Description,
			Required:      c.isRequired,
			Optional:      c.isOptional,
			Computed:      c.isComputed,
			PlanModifiers: mod,
		}
	case types.BoolType:
		mod := []planmodifier.Bool{}
		if c.useStateForUnknown {
			mod = append(mod, boolplanmodifier.UseStateForUnknown())
		}
		if c.requiresReplaceWhenChanged {
			mod = append(mod, boolplanmodifier.RequiresReplace())
		}
		return schema.BoolAttribute{
			Description:   value.Description,
			Required:      c.isRequired,
			Optional:      c.isOptional,
			Computed:      c.isComputed,
			PlanModifiers: mod,
		}
	case types.ListType{}:
		mod := []planmodifier.List{}
		if c.useStateForUnknown {
			mod = append(mod, listplanmodifier.UseStateForUnknown())
		}
		if c.requiresReplaceWhenChanged {
			mod = append(mod, listplanmodifier.RequiresReplace())
		}
		et := getTerraformAttributeType((*mgcSdk.Schema)(value.Items.Value))
		if et == nil {
			// TODO: Error
			return nil
		}
		return schema.ListAttribute{
			Description:   value.Description,
			Required:      c.isRequired,
			Optional:      c.isOptional,
			Computed:      c.isComputed,
			PlanModifiers: mod,
			ElementType:   et,
		}
	default:
		return nil
	}
}

func kebabToSnakeCase(n string) string {
	return strcase.SnakeCase(n)
}
