package provider

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func applyMgcMapToTFState(ctx context.Context, mgcMap map[string]any, attrInfoMap resAttrInfoMap, tfState *tfsdk.State) Diagnostics {
	resInfo := &resAttrInfo{
		tfName:          "tfState",
		mgcName:         "tfState",
		childAttributes: attrInfoMap,
	}
	return applyMgcMap(ctx, mgcMap, resInfo, tfState, path.Empty())
}

func applyMgcMap(ctx context.Context, mgcMap map[string]any, attr *resAttrInfo, tfState *tfsdk.State, path path.Path) Diagnostics {
	diagnostics := Diagnostics{}
	tflog.Debug(
		ctx,
		"[applier] will apply as map",
		map[string]any{"mgcName": attr.mgcName, "tfName": attr.tfName, "value": mgcMap},
	)
	for mgcName, attr := range attr.childAttributes {
		mgcValue, ok := mgcMap[string(mgcName)]
		if !ok {
			continue
		}

		tflog.Debug(
			ctx,
			"[applier] will try to apply map property",
			map[string]any{"propMgcName": mgcName, "propMgcValue": mgcValue},
		)

		tflog.Debug(ctx, fmt.Sprintf("applying %q attribute in state", mgcName), map[string]any{"value": mgcValue})

		attrPath := path.AtName(string(attr.tfName))
		d := applyValueToState(ctx, mgcValue, attr, tfState, attrPath)

		if diagnostics.AppendCheckError(d...) {
			attrSchema, _ := tfState.Schema.AttributeAtPath(ctx, attrPath)
			diagnostics.AddLocalAttributeError(
				attrPath,
				"unable to load value",
				fmt.Sprintf("path: %#v - value: %#v - tfschema: %#v", attrPath, mgcValue, attrSchema),
			)
			return diagnostics
		}
	}
	return diagnostics
}

func applyMgcList(ctx context.Context, mgcValue any, attr *resAttrInfo, tfState *tfsdk.State, path path.Path) Diagnostics {
	diagnostics := Diagnostics{}
	attr = attr.childAttributes["0"]

	// This shouldn't happen, probably, but sometimes the Services return null values for non-nullable values
	if mgcValue == nil {
		d := tfState.SetAttribute(ctx, path, []any{})
		return diagnostics.AppendReturn(Diagnostics(d).DemoteErrorsToWarnings()...)
	}

	mgcList, ok := mgcValue.([]any)
	if !ok {
		diagnostics.AppendReturn(NewLocalErrorDiagnostic(
			fmt.Sprintf("Unable to apply list property %q to State, value is not list", attr.tfName),
			fmt.Sprintf("Property value received from service was not a list: %#v", mgcValue),
		))
	}

	if len(mgcList) == 0 {
		d := tfState.SetAttribute(ctx, path, []any{})
		return diagnostics.AppendReturn(Diagnostics(d).DemoteErrorsToWarnings()...)
	}

	for i, mgcValue := range mgcList {
		attrPath := path.AtListIndex(i)
		d := applyValueToState(ctx, mgcValue, attr, tfState, attrPath)
		if diagnostics.AppendCheckError(d...) {
			attrSchema, _ := tfState.Schema.AttributeAtPath(ctx, attrPath)
			diagnostics.AddLocalAttributeError(attrPath, "unable to load value", fmt.Sprintf("path: %#v - value: %#v - tfschema: %#v", attrPath, mgcValue, attrSchema))
			return diagnostics
		}
	}

	return diagnostics
}

func applyValueToState(ctx context.Context, mgcValue any, attr *resAttrInfo, tfState *tfsdk.State, path path.Path) Diagnostics {
	tflog.Debug(
		ctx,
		"[applier] starting applying mgc value to TF state",
		map[string]any{"mgcName": attr.mgcName, "tfName": attr.tfName, "value": mgcValue},
	)

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
		return applyMgcList(ctx, mgcValue, attr, tfState, path)

	case "object":
		tflog.Debug(ctx, fmt.Sprintf("populating nested object in state at path %#v", path))
		return applyMgcMap(ctx, mgcValue.(map[string]any), attr, tfState, path)

	default:
		// Should this be a local error? Does TF know it already, since it's their function?
		d := tfState.SetAttribute(ctx, path, rv.Interface())
		return Diagnostics(d)
	}
}
