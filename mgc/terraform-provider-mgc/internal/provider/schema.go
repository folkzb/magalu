package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/numberplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stoewer/go-strcase"
	"golang.org/x/exp/slices"
	mgcSdk "magalu.cloud/sdk"
)

type mgcName string
type tfName string

type attribute struct {
	tfName     tfName
	mgcName    mgcName
	mgcSchema  *mgcSdk.Schema
	tfSchema   schema.Attribute
	attributes mgcAttributes
}

type mgcAttributes map[mgcName]*attribute

// Similar schemas are those with the same type and, depending on the type,
// similar properties or restrictions.
func checkSimilarJsonSchemas(a, b *mgcSdk.Schema) bool {
	if a == b {
		return true
	}

	tA, err := getJsonType(a)
	if err != nil {
		return false
	}

	tB, err := getJsonType(b)
	if err != nil {
		return false
	}

	if tA != tB {
		return false
	}

	switch tA {
	default:
		return true
	case "string":
		// Relax if one of them doesn't specify a format
		return a.Format == b.Format || a.Format == "" || b.Format == ""
	case "array":
		return checkSimilarJsonSchemas((*mgcSdk.Schema)(a.Items.Value), (*mgcSdk.Schema)(b.Items.Value))
	case "object":
		// TODO: should we compare Required? I don't think so, but it may be a problem
		if len(a.Properties) != len(b.Properties) {
			return false
		}
		for k, refA := range a.Properties {
			refB := b.Properties[k]
			if refB == nil {
				return false
			}
			if !checkSimilarJsonSchemas((*mgcSdk.Schema)(refA.Value), (*mgcSdk.Schema)(refB.Value)) {
				return false
			}
		}
		// TODO: handle additionalProperties and property patterns
		return true
	}
}

type attributeModifiers struct {
	isRequired                 bool
	isOptional                 bool
	isComputed                 bool
	useStateForUnknown         bool
	requiresReplaceWhenChanged bool
}

func addMgcSchemaAttributes(
	attributes mgcAttributes,
	mgcSchema *mgcSdk.Schema,
	getModifiers func(mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers,
	resourceName string,
	ctx context.Context,
) error {
	for k, ref := range mgcSchema.Properties {
		mgcName := mgcName(k)
		mgcSchema := (*mgcSdk.Schema)(ref.Value)
		if ca, ok := attributes[mgcName]; ok {
			if !checkSimilarJsonSchemas(ca.mgcSchema, mgcSchema) {
				// Ignore update value in favor of create value (This is probably a bug with the API)
				// TODO: Ignore default values when verifying equality
				tflog.Error(ctx, fmt.Sprintf("[resource] schema for `%s`: ignoring DIFFERENT attribute `%s`:\nOLD=%+v\nNEW=%+v", resourceName, k, ca.mgcSchema, mgcSchema))
				continue
			}
			tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: ignoring already computed attribute `%s` ", resourceName, k))
			continue
		}

		tfSchema, err := mgcToTFSchema(mgcSchema, getModifiers(mgcSchema, mgcName))
		if err != nil {
			tflog.Error(ctx, fmt.Sprintf("[resource] schema for %q attribute %q schema: %+v; error: %s", resourceName, k, mgcSchema, err))
			return fmt.Errorf("attribute %q, error=%s", k, err)
		}

		attr := &attribute{
			tfName:    tfNameFromMgc(mgcName),
			mgcName:   mgcName,
			mgcSchema: mgcSchema,
			tfSchema:  tfSchema,
		}
		attributes[mgcName] = attr
		tflog.Debug(ctx, fmt.Sprintf("[resource] schema for %q attribute %q: %+v", resourceName, k, attr))
	}

	return nil
}

func (r *MgcResource) getCreateParamsModifiers(mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	k := string(mgcName)
	isRequired := slices.Contains(mgcSchema.Required, k)
	return attributeModifiers{
		isRequired:                 isRequired,
		isOptional:                 !isRequired,
		isComputed:                 !isRequired && r.read.ResultSchema().Properties[k] != nil, // If not required and present in read it can be compute
		useStateForUnknown:         false,
		requiresReplaceWhenChanged: r.update.ParametersSchema().Properties[k] == nil,
	}
}

func (r *MgcResource) getUpdateParamsModifiers(mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	k := string(mgcName)
	isCreated := r.create.ResultSchema().Properties[k] != nil
	required := slices.Contains(mgcSchema.Required, k)
	return attributeModifiers{
		isRequired:                 required && !isCreated,
		isOptional:                 !required && !isCreated,
		isComputed:                 !required || isCreated,
		useStateForUnknown:         true,
		requiresReplaceWhenChanged: false,
	}
}

func (r *MgcResource) getResultModifiers(mgcSchema *mgcSdk.Schema, mgcName mgcName) attributeModifiers {
	return attributeModifiers{
		isRequired:                 false,
		isOptional:                 false,
		isComputed:                 true,
		useStateForUnknown:         true,
		requiresReplaceWhenChanged: false,
	}
}

func (r *MgcResource) readInputAttributes(ctx context.Context) diag.Diagnostics {
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

	r.inputAttr = input
	return d
}

func (r *MgcResource) readOutputAttributes(ctx context.Context) diag.Diagnostics {
	d := diag.Diagnostics{}
	if len(r.outputAttr) != 0 {
		return d
	}
	tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: reading output attributes", r.name))

	output := mgcAttributes{}
	err := addMgcSchemaAttributes(
		output,
		r.create.ResultSchema(),
		r.getResultModifiers,
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
		r.getResultModifiers,
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

func (r *MgcResource) generateTFSchema(ctx context.Context) (tfSchema schema.Schema, d diag.Diagnostics) {
	var tfsa map[tfName]schema.Attribute
	tfsa, d = r.generateTFAttributes(ctx)
	if d.HasError() {
		return
	}

	tfSchema = schema.Schema{Attributes: map[string]schema.Attribute{}}
	for tfName, tfAttr := range tfsa {
		tfSchema.Attributes[string(tfName)] = tfAttr
	}
	return
}

func (r *MgcResource) generateTFAttributes(ctx context.Context) (tfa map[tfName]schema.Attribute, d diag.Diagnostics) {
	d.Append(r.readInputAttributes(ctx)...)
	if d.HasError() {
		return
	}
	d.Append(r.readOutputAttributes(ctx)...)
	if d.HasError() {
		return
	}

	tfa = map[tfName]schema.Attribute{}
	for name, iattr := range r.inputAttr {
		// Split attributes that differ between input/output
		if oattr := r.outputAttr[name]; oattr != nil {
			if !checkSimilarJsonSchemas(oattr.mgcSchema, iattr.mgcSchema) {
				os, _ := oattr.mgcSchema.MarshalJSON()
				is, _ := iattr.mgcSchema.MarshalJSON()
				tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: attribute `%s` differs between input and output. input: %s - output %s", r.name, name, is, os))
				iattr.tfName = iattr.tfName.asDesired()
				oattr.tfName = iattr.tfName.asCurrent()
			}
		}

		tfa[iattr.tfName] = iattr.tfSchema
	}

	for _, oattr := range r.outputAttr {
		// If they don't differ and it's already created skip
		if _, ok := tfa[oattr.tfName]; ok {
			continue
		}

		tfa[oattr.tfName] = oattr.tfSchema
	}

	return
}

func mgcToTFSchema(mgcSchema *mgcSdk.Schema, m attributeModifiers) (schema.Attribute, error) {
	// TODO: Handle default values

	t, err := getJsonType(mgcSchema)
	if err != nil {
		return nil, err
	}
	description := mgcSchema.Description

	switch t {
	case "string":
		// I wanted to use an interface to define the modifiers regardless of the attr type
		// but couldn't find the interface, it seems everything is redefined for each type
		// https://github.com/hashicorp/terraform-plugin-framework/blob/main/internal/fwschema/fwxschema/attribute_plan_modification.go
		mod := []planmodifier.String{}
		if m.useStateForUnknown {
			mod = append(mod, stringplanmodifier.UseStateForUnknown())
		}
		if m.requiresReplaceWhenChanged {
			mod = append(mod, stringplanmodifier.RequiresReplace())
		}
		return schema.StringAttribute{
			Description:   description,
			Required:      m.isRequired,
			Optional:      m.isOptional,
			Computed:      m.isComputed,
			PlanModifiers: mod,
		}, nil
	case "number":
		mod := []planmodifier.Number{}
		if m.useStateForUnknown {
			mod = append(mod, numberplanmodifier.UseStateForUnknown())
		}
		if m.requiresReplaceWhenChanged {
			mod = append(mod, numberplanmodifier.RequiresReplace())
		}
		return schema.NumberAttribute{
			Description:   description,
			Required:      m.isRequired,
			Optional:      m.isOptional,
			Computed:      m.isComputed,
			PlanModifiers: mod,
		}, nil
	case "integer":
		mod := []planmodifier.Int64{}
		if m.useStateForUnknown {
			mod = append(mod, int64planmodifier.UseStateForUnknown())
		}
		if m.requiresReplaceWhenChanged {
			mod = append(mod, int64planmodifier.RequiresReplace())
		}
		return schema.Int64Attribute{
			Description:   description,
			Required:      m.isRequired,
			Optional:      m.isOptional,
			Computed:      m.isComputed,
			PlanModifiers: mod,
		}, nil
	case "boolean":
		mod := []planmodifier.Bool{}
		if m.useStateForUnknown {
			mod = append(mod, boolplanmodifier.UseStateForUnknown())
		}
		if m.requiresReplaceWhenChanged {
			mod = append(mod, boolplanmodifier.RequiresReplace())
		}
		return schema.BoolAttribute{
			Description:   description,
			Required:      m.isRequired,
			Optional:      m.isOptional,
			Computed:      m.isComputed,
			PlanModifiers: mod,
		}, nil
	case "array":
		return nil, fmt.Errorf("array not supported yet")
	case "object":
		return nil, fmt.Errorf("object not supported yet")
	default:
		return nil, fmt.Errorf("type %q not supported", t)
	}
}

func tfNameFromMgc(n mgcName) tfName {
	return tfName(strcase.SnakeCase(string(n)))
}

func (n tfName) asDesired() tfName {
	return "desired_" + n
}

func (n tfName) asCurrent() tfName {
	return "current_" + n
}
