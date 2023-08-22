package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"magalu.cloud/core"
)

var states = []tftypes.Value{
	tftypes.NewValue(tftypes.String, "test_string"),
	tftypes.NewValue(tftypes.Bool, true),
	tftypes.NewValue(tftypes.Number, 10),
	tftypes.NewValue(tftypes.Number, 10),
	tftypes.NewValue(tftypes.Number, 10.0),
	tftypes.NewValue(tftypes.Number, 0.000000000000000000000000001),
	tftypes.NewValue(
		tftypes.List{ElementType: tftypes.String},
		[]tftypes.Value{tftypes.NewValue(tftypes.String, "zero"), tftypes.NewValue(tftypes.String, "one")},
	),

	tftypes.NewValue(
		tftypes.List{
			ElementType: tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"value": tftypes.String,
				},
			},
		},
		[]tftypes.Value{
			tftypes.NewValue(
				tftypes.Object{
					AttributeTypes: map[string]tftypes.Type{
						"value": tftypes.String,
					},
				},
				map[string]tftypes.Value{
					"value": tftypes.NewValue(tftypes.String, "myvalueresult"),
				},
			),
		},
	),

	tftypes.NewValue(
		tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"value": tftypes.Object{
					AttributeTypes: map[string]tftypes.Type{
						"value_nested": tftypes.String,
					},
				},
			},
		},
		map[string]tftypes.Value{
			"value": tftypes.NewValue(
				tftypes.Object{
					AttributeTypes: map[string]tftypes.Type{
						"value_nested": tftypes.String,
					},
				},
				map[string]tftypes.Value{
					"value_nested": tftypes.NewValue(tftypes.String, "myvalueresult"),
				},
			),
		},
	),

	tftypes.NewValue(
		tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"value": tftypes.Object{
					AttributeTypes: map[string]tftypes.Type{
						"tf_value_nested": tftypes.String,
					},
				},
			},
		},
		map[string]tftypes.Value{
			"value": tftypes.NewValue(
				tftypes.Object{
					AttributeTypes: map[string]tftypes.Type{
						"tf_value_nested": tftypes.String,
					},
				},
				map[string]tftypes.Value{
					"tf_value_nested": tftypes.NewValue(tftypes.String, "myvalueresult"),
				},
			),
		},
	),

	tftypes.NewValue(
		tftypes.List{
			ElementType: tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"tf_value": tftypes.String,
				},
			},
		},
		[]tftypes.Value{
			tftypes.NewValue(
				tftypes.Object{
					AttributeTypes: map[string]tftypes.Type{
						"tf_value": tftypes.String,
					},
				},
				map[string]tftypes.Value{
					"tf_value": tftypes.NewValue(tftypes.String, "myvalueresult"),
				},
			),
		},
	),
}

var schemas = []*core.Schema{
	core.NewStringSchema(),
	core.NewBooleanSchema(),
	core.NewIntegerSchema(),
	core.NewNumberSchema(),
	core.NewNumberSchema(),
	core.NewNumberSchema(),
	core.NewArraySchema(core.NewStringSchema()),
	core.NewArraySchema(
		core.NewObjectSchema(map[string]*core.Schema{
			"value": core.NewStringSchema(),
		}, []string{"value"}),
	),
	core.NewObjectSchema(map[string]*core.Schema{
		"value": core.NewObjectSchema(map[string]*core.Schema{
			"value_nested": core.NewStringSchema(),
		}, []string{"value_nested"}),
	}, []string{"value"}),
	core.NewObjectSchema(map[string]*core.Schema{
		"value": core.NewObjectSchema(map[string]*core.Schema{
			"value_nested": core.NewStringSchema(),
		}, []string{"value_nested"}),
	}, []string{"value"}),
	core.NewArraySchema(
		core.NewObjectSchema(map[string]*core.Schema{
			"value": core.NewStringSchema(),
		}, []string{"value"}),
	),
}

var results = []any{
	"test_string",
	true,
	(int64)(10),
	(float64)(10),
	(float64)(10.0),
	(float64)(0.000000000000000000000000001),
	[]any{"zero", "one"},
	[]any{map[string]any{"value": "myvalueresult"}},
	map[string]any{"value": map[string]any{"value_nested": "myvalueresult"}},
	map[string]any{"value": map[string]any{"value_nested": "myvalueresult"}},
	[]any{map[string]any{"value": "myvalueresult"}},
}

var attrInfo = map[string]*attribute{
	"value": {
		name: "value",
		attributes: map[string]*attribute{
			"value_nested": {
				name: "value_nested",
			},
		},
	},
}
var attrInfoList = map[string]*attribute{
	"0": {
		name: "0",
		attributes: map[string]*attribute{
			"value": {
				name: "value",
			},
		},
	},
}
var attrInfoTFNameObjectNested = map[string]*attribute{
	"value": {
		name: "value",
		attributes: map[string]*attribute{
			"value_nested": {
				name: "tf_value_nested",
			},
		},
	},
}
var attrInfoTFNameObjectInList = map[string]*attribute{
	"0": {
		name: "0",
		attributes: map[string]*attribute{
			"value": {
				name: "tf_value",
			},
		},
	},
}

var attrInfos = []map[string]*attribute{
	{},
	{},
	{},
	{},
	{},
	{},
	{"0": {}},
	attrInfoList,
	attrInfo,
	attrInfoTFNameObjectNested,
	attrInfoTFNameObjectInList,
}

func TestConvertTFToValue(t *testing.T) {
	conv := converter{
		ctx:  context.Background(),
		diag: diag.Diagnostics{},
	}

	for i := 0; i < len(states); i++ {
		result := conv.convertTFToValue(schemas[i], attrInfos[i], states[i])
		if !reflect.DeepEqual(result, results[i]) {
			t.Fatalf("result differs from expected: %T:%+v %T:%+v %+v", result, result, results[i], results[i], conv.diag)
		}
	}
}
