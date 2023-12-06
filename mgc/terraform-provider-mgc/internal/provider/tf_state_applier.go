package provider

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type tfStateApplier struct {
	ctx      context.Context
	diag     *diag.Diagnostics
	tfSchema *schema.Schema
}

func newTFStateApplier(ctx context.Context, diag *diag.Diagnostics, tfSchema *schema.Schema) tfStateApplier {
	return tfStateApplier{
		ctx:      ctx,
		diag:     diag,
		tfSchema: tfSchema,
	}
}

func (c *tfStateApplier) applyMgcMap(mgcMap map[string]any, attributes resAttrInfoMap, ctx context.Context, tfState *tfsdk.State, path path.Path) {
	for mgcName, attr := range attributes {
		mgcValue, ok := mgcMap[string(mgcName)]
		if !ok {
			continue
		}

		tflog.Debug(ctx, fmt.Sprintf("applying %q attribute in state", mgcName), map[string]any{"value": mgcValue})

		attrPath := path.AtName(string(attr.tfName))
		c.applyValueToState(mgcValue, attr, ctx, tfState, attrPath)

		if c.diag.HasError() {
			attrSchema, _ := tfState.Schema.AttributeAtPath(ctx, attrPath)
			c.diag.AddAttributeError(
				attrPath,
				"unable to load value",
				fmt.Sprintf("path: %#v - value: %#v - tfschema: %#v", attrPath, mgcValue, attrSchema),
			)
			return
		}
	}
}

func (c *tfStateApplier) applyMgcList(mgcList []any, attributes resAttrInfoMap, ctx context.Context, tfState *tfsdk.State, path path.Path) {
	attr := attributes["0"]

	for i, mgcValue := range mgcList {
		attrPath := path.AtListIndex(i)
		c.applyValueToState(mgcValue, attr, ctx, tfState, attrPath)

		if c.diag.HasError() {
			attrSchema, _ := tfState.Schema.AttributeAtPath(ctx, attrPath)
			c.diag.AddAttributeError(attrPath, "unable to load value", fmt.Sprintf("path: %#v - value: %#v - tfschema: %#v", attrPath, mgcValue, attrSchema))
			return
		}
	}
}

func (c *tfStateApplier) applyValueToState(mgcValue any, attr *resAttrInfo, ctx context.Context, tfState *tfsdk.State, path path.Path) {
	rv := reflect.ValueOf(mgcValue)
	if mgcValue == nil {
		// We must check the nil value type, since SetAttribute method requires a typed nil
		switch attr.mgcSchema.Type {
		case "string":
			rv = reflect.ValueOf((*string)(nil))
		case "integer":
			rv = reflect.ValueOf((*int64)(nil))
		case "number":
			rv = reflect.ValueOf((*float64)(nil))
		case "boolean":
			rv = reflect.ValueOf((*bool)(nil))
		}
	}

	switch attr.mgcSchema.Type {
	case "array":
		tflog.Debug(ctx, fmt.Sprintf("populating list in state at path %#v", path))
		c.applyMgcList(mgcValue.([]any), attr.childAttributes, ctx, tfState, path)

	case "object":
		tflog.Debug(ctx, fmt.Sprintf("populating nested object in state at path %#v", path))
		c.applyMgcMap(mgcValue.(map[string]any), attr.childAttributes, ctx, tfState, path)

	default:
		c.diag.Append(tfState.SetAttribute(ctx, path, rv.Interface())...)
	}
}
