package provider

import (
	"context"
	"fmt"
	"reflect"
	"regexp"

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
	isID       bool // TODO: remove this once we use links, see https://github.com/profusion/magalu/issues/215
}

type mgcAttributes map[mgcName]*attribute

var idRexp = regexp.MustCompile(`(^id$|_id$)`)

func (r *MgcResource) readInputAttributes(ctx context.Context) diag.Diagnostics {
	d := diag.Diagnostics{}
	if len(r.inputAttr) != 0 {
		return d
	}
	tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: reading input attributes", r.name))

	input := mgcAttributes{}
	cschema := r.create.ParametersSchema()
	for attr, ref := range cschema.Properties {
		required := slices.Contains(cschema.Required, attr)
		mgcName := mgcName(attr)
		mgcSchema := (*mgcSdk.Schema)(ref.Value)
		isRequired := required
		isOptional := !required
		isComputed := !required && r.read.ResultSchema().Properties[attr] != nil // If not required and present in read it can be computed
		useStateForUnknown := false
		requiresReplaceWhenChanged := r.update.ParametersSchema().Properties[attr] == nil
		tfSchema, err := mgcToTFSchema(mgcSchema, isRequired, isOptional, isComputed, useStateForUnknown, requiresReplaceWhenChanged)
		if err != nil {
			d.AddError("could not create TF schema", fmt.Sprintf("attribute %q, error=%s", attr, err))
			continue
		}

		input[mgcName] = &attribute{
			tfName:    tfNameFromMgc(mgcName),
			mgcName:   mgcName,
			mgcSchema: mgcSchema,
			tfSchema:  tfSchema,
			isID:      false,
		}
		tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: attribute `%s` info - %+v", r.name, attr, input[mgcName]))
	}

	uschema := r.update.ParametersSchema()
	hasID := uschema.Properties["id"]
	for attr, ref := range uschema.Properties {
		mgcName := mgcName(attr)
		mgcSchema := (*mgcSdk.Schema)(ref.Value)
		if ca, ok := input[mgcName]; ok {
			if !reflect.DeepEqual(ca.mgcSchema, mgcSchema) {
				// Ignore update value in favor of create value (This is probably a bug with the API)
				// TODO: Ignore default values when verifying equality
				// TODO: Don't forget to add the path when using recursion
				// err := fmt.Sprintf("[resource] schema for `%s`: input attribute `%s` is different between create and update - create: %+v - update: %+v ", r.name, attr, ca.schema, us)
				// d.AddError("Attribute schema is different between create and update schemas", err)
				continue
			}
			tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: ignoring already computed attribute `%s` ", r.name, attr))
			continue
		}

		// TODO: Better handle ID attributions
		// Consider other ID elements as ID if "id" doesn't exist
		isID := false
		if hasID != nil {
			isID = attr == "id"
		} else {
			isID = idRexp.MatchString(attr)
		}

		required := slices.Contains(uschema.Required, attr)
		isRequired := required && !isID
		isOptional := !required && !isID
		isComputed := !required || isID
		useStateForUnknown := true
		requiresReplaceWhenChanged := false
		tfSchema, err := mgcToTFSchema(mgcSchema, isRequired, isOptional, isComputed, useStateForUnknown, requiresReplaceWhenChanged)
		if err != nil {
			d.AddError("could not create TF schema", fmt.Sprintf("attribute %q, error=%s", attr, err))
			continue
		}

		input[mgcName] = &attribute{
			tfName:    tfNameFromMgc(mgcName),
			mgcName:   mgcName,
			mgcSchema: mgcSchema,
			tfSchema:  tfSchema,
			isID:      isID,
		}
		tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: attribute `%s` info - %+v", r.name, attr, input[mgcName]))
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
	crschema := r.create.ResultSchema()
	hasID := crschema.Properties["id"]
	for attr, ref := range crschema.Properties {
		mgcName := mgcName(attr)
		mgcSchema := (*mgcSdk.Schema)(ref.Value)
		isID := false
		if hasID != nil {
			isID = attr == "id"
		} else {
			isID = idRexp.MatchString(attr)
		}
		isRequired := false
		isOptional := false
		isComputed := true
		useStateForUnknown := true
		requiresReplaceWhenChanged := false // This one is useless in this case
		tfSchema, err := mgcToTFSchema(mgcSchema, isRequired, isOptional, isComputed, useStateForUnknown, requiresReplaceWhenChanged)
		if err != nil {
			d.AddError("could not create TF schema", fmt.Sprintf("attribute %q, error=%s", attr, err))
			continue
		}

		output[mgcName] = &attribute{
			tfName:    tfNameFromMgc(mgcName),
			mgcName:   mgcName,
			mgcSchema: mgcSchema,
			tfSchema:  tfSchema,
			isID:      isID,
		}
		tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: attribute `%s` info - %+v", r.name, attr, output[mgcName]))
	}

	for attr, ref := range r.read.ResultSchema().Properties {
		mgcName := mgcName(attr)
		mgcSchema := (*mgcSdk.Schema)(ref.Value)
		if ra, ok := output[mgcName]; ok {
			if !reflect.DeepEqual(ra.mgcSchema, mgcSchema) {
				// Ignore read value in favor of create result value (This is probably a bug with the API)
				// err := fmt.Sprintf("[resource] schema for `%s`: output attribute `%s` is different between create result and read - create result: %+v - read: %+v ", r.name, attr, ra.schema, rs)
				// d.AddError("Attribute schema is different between create result and read schemas", err)
				continue
			}
			tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: ignoring already computed attribute `%s` ", r.name, attr))
			continue
		}
		isRequired := false
		isOptional := false
		isComputed := true
		useStateForUnknown := true
		requiresReplaceWhenChanged := false // This one is useless in this case
		tfSchema, err := mgcToTFSchema(mgcSchema, isRequired, isOptional, isComputed, useStateForUnknown, requiresReplaceWhenChanged)
		if err != nil {
			d.AddError("could not create TF schema", fmt.Sprintf("attribute %q, error=%s", attr, err))
			continue
		}

		output[mgcName] = &attribute{
			tfName:    tfNameFromMgc(mgcName),
			mgcName:   mgcName,
			mgcSchema: mgcSchema,
			tfSchema:  tfSchema,
			isID:      false,
		}
		tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: attribute `%s` info - %+v", r.name, attr, output[mgcName]))
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
		if oattr := r.outputAttr[name]; oattr != nil && !iattr.isID {
			if !reflect.DeepEqual(oattr.mgcSchema, iattr.mgcSchema) {
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

func mgcToTFSchema(mgcSchema *mgcSdk.Schema, isRequired bool, isOptional bool, isComputed bool, useStateForUnknown bool, requiresReplaceWhenChanged bool) (schema.Attribute, error) {
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
		if useStateForUnknown {
			mod = append(mod, stringplanmodifier.UseStateForUnknown())
		}
		if requiresReplaceWhenChanged {
			mod = append(mod, stringplanmodifier.RequiresReplace())
		}
		return schema.StringAttribute{
			Description:   description,
			Required:      isRequired,
			Optional:      isOptional,
			Computed:      isComputed,
			PlanModifiers: mod,
		}, nil
	case "number":
		mod := []planmodifier.Number{}
		if useStateForUnknown {
			mod = append(mod, numberplanmodifier.UseStateForUnknown())
		}
		if requiresReplaceWhenChanged {
			mod = append(mod, numberplanmodifier.RequiresReplace())
		}
		return schema.NumberAttribute{
			Description:   description,
			Required:      isRequired,
			Optional:      isOptional,
			Computed:      isComputed,
			PlanModifiers: mod,
		}, nil
	case "integer":
		mod := []planmodifier.Int64{}
		if useStateForUnknown {
			mod = append(mod, int64planmodifier.UseStateForUnknown())
		}
		if requiresReplaceWhenChanged {
			mod = append(mod, int64planmodifier.RequiresReplace())
		}
		return schema.Int64Attribute{
			Description:   description,
			Required:      isRequired,
			Optional:      isOptional,
			Computed:      isComputed,
			PlanModifiers: mod,
		}, nil
	case "boolean":
		mod := []planmodifier.Bool{}
		if useStateForUnknown {
			mod = append(mod, boolplanmodifier.UseStateForUnknown())
		}
		if requiresReplaceWhenChanged {
			mod = append(mod, boolplanmodifier.RequiresReplace())
		}
		return schema.BoolAttribute{
			Description:   description,
			Required:      isRequired,
			Optional:      isOptional,
			Computed:      isComputed,
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
