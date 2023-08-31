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

type attribute struct {
	tfName                     string
	mgcSchema                  *mgcSdk.Schema
	attributes                 map[string]*attribute
	isID                       bool
	isRequired                 bool
	isOptional                 bool
	isComputed                 bool
	useStateForUnknown         bool
	requiresReplaceWhenChanged bool
}

var idRexp = regexp.MustCompile(`(^id$|_id$)`)

func (r *MgcResource) readInputAttributes(ctx context.Context) diag.Diagnostics {
	d := diag.Diagnostics{}
	if len(r.inputAttr) != 0 {
		return d
	}
	tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: reading input attributes", r.name))

	input := map[string]*attribute{}
	cschema := r.create.ParametersSchema()
	for attr, ref := range cschema.Properties {
		required := slices.Contains(cschema.Required, attr)
		input[attr] = &attribute{
			tfName:                     kebabToSnakeCase(attr),
			mgcSchema:                  (*mgcSdk.Schema)(ref.Value),
			isID:                       false,
			isRequired:                 required,
			isOptional:                 !required,
			isComputed:                 !required && r.read.ResultSchema().Properties[attr] != nil, // If not required and present in read it can be computed
			useStateForUnknown:         false,
			requiresReplaceWhenChanged: r.update.ParametersSchema().Properties[attr] == nil,
		}
		tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: attribute `%s` info - %+v", r.name, attr, input[attr]))
	}

	uschema := r.update.ParametersSchema()
	hasID := uschema.Properties["id"]
	for attr, ref := range uschema.Properties {
		if ca, ok := input[attr]; ok {
			us := ref.Value
			if !reflect.DeepEqual(ca.mgcSchema, (*mgcSdk.Schema)(us)) {
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
		input[attr] = &attribute{
			tfName:                     kebabToSnakeCase(attr),
			mgcSchema:                  (*mgcSdk.Schema)(ref.Value),
			isID:                       isID,
			isRequired:                 required && !isID,
			isOptional:                 !required && !isID,
			isComputed:                 !required || isID,
			useStateForUnknown:         true,
			requiresReplaceWhenChanged: false,
		}
		tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: attribute `%s` info - %+v", r.name, attr, input[attr]))
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

	output := map[string]*attribute{}
	crschema := r.create.ResultSchema()
	hasID := crschema.Properties["id"]
	for attr, ref := range crschema.Properties {
		isID := false
		if hasID != nil {
			isID = attr == "id"
		} else {
			isID = idRexp.MatchString(attr)
		}
		output[attr] = &attribute{
			tfName:                     kebabToSnakeCase(attr),
			mgcSchema:                  (*mgcSdk.Schema)(ref.Value),
			isID:                       isID,
			isRequired:                 false,
			isOptional:                 false,
			isComputed:                 true,
			useStateForUnknown:         true,
			requiresReplaceWhenChanged: false, // This one is useless in this case
		}
		tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: attribute `%s` info - %+v", r.name, attr, output[attr]))
	}

	for attr, ref := range r.read.ResultSchema().Properties {
		if ra, ok := output[attr]; ok {
			rs := ref.Value
			if !reflect.DeepEqual(ra.mgcSchema, (*mgcSdk.Schema)(rs)) {
				// Ignore read value in favor of create result value (This is probably a bug with the API)
				// err := fmt.Sprintf("[resource] schema for `%s`: output attribute `%s` is different between create result and read - create result: %+v - read: %+v ", r.name, attr, ra.schema, rs)
				// d.AddError("Attribute schema is different between create result and read schemas", err)
				continue
			}
			tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: ignoring already computed attribute `%s` ", r.name, attr))
			continue
		}

		output[attr] = &attribute{
			tfName:                     kebabToSnakeCase(attr),
			mgcSchema:                  (*mgcSdk.Schema)(ref.Value),
			isID:                       false,
			isRequired:                 false,
			isOptional:                 false,
			isComputed:                 true,
			useStateForUnknown:         true,
			requiresReplaceWhenChanged: false, // This one is useless in this case
		}
		tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: attribute `%s` info - %+v", r.name, attr, output[attr]))
	}

	r.outputAttr = output
	return d
}

func (r *MgcResource) generateTFAttributes(ctx context.Context) (*map[string]schema.Attribute, diag.Diagnostics) {
	d := diag.Diagnostics{}
	d.Append(r.readInputAttributes(ctx)...)
	d.Append(r.readOutputAttributes(ctx)...)

	tfa := map[string]schema.Attribute{}
	for name, iattr := range r.inputAttr {
		// Split attributes that differ between input/output
		if oattr := r.outputAttr[name]; oattr != nil && !iattr.isID {
			if !reflect.DeepEqual(oattr.mgcSchema, iattr.mgcSchema) {
				os, _ := oattr.mgcSchema.MarshalJSON()
				is, _ := iattr.mgcSchema.MarshalJSON()
				tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: attribute `%s` differs between input and output. input: %s - output %s", r.name, name, is, os))
				iattr.tfName = kebabToSnakeCase("desired_" + iattr.tfName)
				oattr.tfName = kebabToSnakeCase("current_" + oattr.tfName)
			}
		}

		at := sdkToTerraformAttribute(ctx, iattr, diag.Diagnostics{})
		// TODO: This shouldn't happen after we handle complex types like slices and objects
		// TODO: Remove debug log
		if at == nil {
			err := fmt.Sprintf("[resource] schema for `%s`: unable to create terraform attribute `%s` - data: %+v", r.name, iattr.tfName, iattr)
			tflog.Debug(ctx, err)
			// TODO: Uncomment the error
			// d.AddError("Unknown attribute type", err)
			continue
		}
		tflog.Debug(ctx, fmt.Sprintf("[resource] schema for `%s`: terraform input attribute `%s` created", r.name, iattr.tfName))
		tfa[iattr.tfName] = at
	}

	for _, oattr := range r.outputAttr {
		// If they don't differ and it's already created skip
		if _, ok := tfa[oattr.tfName]; ok {
			continue
		}

		at := sdkToTerraformAttribute(ctx, oattr, diag.Diagnostics{})
		if at == nil {
			// TODO: This shouldn't happen after we handle complex types like slices and objects
			// TODO: Remove debug log
			err := fmt.Sprintf("[resource] schema for `%s`: unable to create terraform attribute `%s` - data: %+v", r.name, oattr.tfName, oattr)
			tflog.Debug(ctx, err)
			// TODO: Uncomment the error
			// d.AddError("Unknown attribute type", err)
			continue
		}
		tfa[oattr.tfName] = at
	}

	return &tfa, d
}

func sdkToTerraformAttribute(ctx context.Context, c *attribute, di diag.Diagnostics) schema.Attribute {
	if c.mgcSchema == nil || c == nil {
		di.AddError("Invalid attribute pointer", fmt.Sprintf("ERROR invalid pointer, attribute pointer is nil %v %v", c.mgcSchema, c))
		return nil
	}

	conv := tfStateConverter{
		ctx:  ctx,
		diag: &di,
	}

	// TODO: Handle default values

	value := c.mgcSchema
	t := conv.getAttributeType(value)
	if di.HasError() {
		return nil
	}

	switch t {
	case "string":
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
	case "number":
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
	case "integer":
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
	case "boolean":
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
	case "array":
		return nil
	case "object":
		return nil
	default:
		return nil
	}
}

func kebabToSnakeCase(n string) string {
	return strcase.SnakeCase(n)
}
