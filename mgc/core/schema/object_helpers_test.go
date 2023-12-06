package schema

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func checkObjectPropertySchemas(t *testing.T, prefix string, expected []ObjectPropertySchema, got []ObjectPropertySchema) {
	if len(expected) != len(got) {
		t.Errorf("%s: expected %d elements, got %d", prefix, len(expected), len(got))
		return
	}

	for i, e := range expected {
		g := got[i]
		if e != g {
			t.Errorf("%s: expected %d to be:\n\t%#v\ngot:\n\t%#v", prefix, i, e, g)
		}
	}
}

func Test_ForEachObjectProperty(t *testing.T) {
	schemaString := NewStringSchema()
	schemaInteger := NewIntegerSchema()
	schemaBoolean := NewBooleanSchema()
	schemaObjectString := NewObjectSchema(map[string]*Schema{
		"propName": schemaString,
		"a":        schemaBoolean,
	}, nil)
	schemaObjectInteger := NewObjectSchema(map[string]*Schema{
		"propName": schemaInteger,
		"x":        schemaBoolean,
	}, nil)
	schemaRoot := NewObjectSchema(map[string]*Schema{
		"rootProp": schemaBoolean,
	}, nil)
	schemaRoot.AnyOf = openapi3.SchemaRefs{
		NewSchemaRef("", schemaObjectString),
		NewSchemaRef("", schemaObjectInteger),
	}
	schemaRoot.OneOf = openapi3.SchemaRefs{
		NewSchemaRef("", schemaObjectInteger), // swapped orders
		NewSchemaRef("", schemaObjectString),
	}

	var got []ObjectPropertySchema

	finished, err := ForEachObjectProperty(schemaRoot, func(ps ObjectPropertySchema) (finished bool, err error) {
		got = append(got, ps)
		return true, nil
	})

	if finished != true {
		t.Error("expected finished = true")
		return
	}
	checkNoError(t, "ForEachObjectProperty", err)

	expected := []ObjectPropertySchema{
		{Container: schemaRoot, Field: "", Index: 0, PropName: "rootProp", PropSchema: schemaBoolean},
		{Container: schemaObjectString, Field: "AnyOf", Index: 0, PropName: "a", PropSchema: schemaBoolean},
		{Container: schemaObjectString, Field: "AnyOf", Index: 0, PropName: "propName", PropSchema: schemaString},
		{Container: schemaObjectInteger, Field: "AnyOf", Index: 1, PropName: "propName", PropSchema: schemaInteger},
		{Container: schemaObjectInteger, Field: "AnyOf", Index: 1, PropName: "x", PropSchema: schemaBoolean},
		{Container: schemaObjectInteger, Field: "OneOf", Index: 0, PropName: "propName", PropSchema: schemaInteger},
		{Container: schemaObjectInteger, Field: "OneOf", Index: 0, PropName: "x", PropSchema: schemaBoolean},
		{Container: schemaObjectString, Field: "OneOf", Index: 1, PropName: "a", PropSchema: schemaBoolean},
		{Container: schemaObjectString, Field: "OneOf", Index: 1, PropName: "propName", PropSchema: schemaString},
	}

	checkObjectPropertySchemas(t, "ForEachObjectProperty", expected, got)
}

func Test_CollectObjectPropertySchemas(t *testing.T) {
	schemaString := NewStringSchema()
	schemaInteger := NewIntegerSchema()
	schemaBoolean := NewBooleanSchema()
	schemaObjectString := NewObjectSchema(map[string]*Schema{
		"propName": schemaString,
		"a":        schemaBoolean,
	}, nil)
	schemaObjectInteger := NewObjectSchema(map[string]*Schema{
		"propName": schemaInteger,
		"x":        schemaBoolean,
	}, nil)
	schemaRoot := NewObjectSchema(map[string]*Schema{
		"propName": schemaBoolean,
	}, nil)
	schemaRoot.AnyOf = openapi3.SchemaRefs{
		NewSchemaRef("", schemaObjectString),
		NewSchemaRef("", schemaObjectInteger),
	}
	schemaRoot.OneOf = openapi3.SchemaRefs{
		NewSchemaRef("", schemaObjectInteger), // swapped orders
		NewSchemaRef("", schemaObjectString),
	}

	got := CollectObjectPropertySchemas(schemaRoot, "propName")
	expected := []ObjectPropertySchema{
		{Container: schemaRoot, Field: "", Index: 0, PropName: "propName", PropSchema: schemaBoolean},
		{Container: schemaObjectString, Field: "AnyOf", Index: 0, PropName: "propName", PropSchema: schemaString},
		{Container: schemaObjectInteger, Field: "AnyOf", Index: 1, PropName: "propName", PropSchema: schemaInteger},
		{Container: schemaObjectInteger, Field: "OneOf", Index: 0, PropName: "propName", PropSchema: schemaInteger},
		{Container: schemaObjectString, Field: "OneOf", Index: 1, PropName: "propName", PropSchema: schemaString},
	}

	checkObjectPropertySchemas(t, "CollectObjectPropertySchemas", expected, got)
}

func Test_CollectAllObjectPropertySchemas(t *testing.T) {
	schemaString := NewStringSchema()
	schemaInteger := NewIntegerSchema()
	schemaBoolean := NewBooleanSchema()
	schemaObjectString := NewObjectSchema(map[string]*Schema{
		"propName": schemaString,
		"a":        schemaBoolean,
	}, nil)
	schemaObjectInteger := NewObjectSchema(map[string]*Schema{
		"propName": schemaInteger,
		"x":        schemaBoolean,
	}, nil)
	schemaRoot := NewObjectSchema(map[string]*Schema{
		"rootProp": schemaBoolean,
	}, nil)
	schemaRoot.AnyOf = openapi3.SchemaRefs{
		NewSchemaRef("", schemaObjectString),
		NewSchemaRef("", schemaObjectInteger),
	}
	schemaRoot.OneOf = openapi3.SchemaRefs{
		NewSchemaRef("", schemaObjectInteger), // swapped orders
		NewSchemaRef("", schemaObjectString),
	}

	got := CollectAllObjectPropertySchemas(schemaRoot)
	expected := map[string][]ObjectPropertySchema{
		"a": {
			{Container: schemaObjectString, Field: "AnyOf", Index: 0, PropName: "a", PropSchema: schemaBoolean},
			{Container: schemaObjectString, Field: "OneOf", Index: 1, PropName: "a", PropSchema: schemaBoolean},
		},
		"rootProp": {
			{Container: schemaRoot, Field: "", Index: 0, PropName: "rootProp", PropSchema: schemaBoolean},
		},
		"x": {
			{Container: schemaObjectInteger, Field: "AnyOf", Index: 1, PropName: "x", PropSchema: schemaBoolean},
			{Container: schemaObjectInteger, Field: "OneOf", Index: 0, PropName: "x", PropSchema: schemaBoolean},
		},
		"propName": {
			{Container: schemaObjectString, Field: "AnyOf", Index: 0, PropName: "propName", PropSchema: schemaString},
			{Container: schemaObjectInteger, Field: "AnyOf", Index: 1, PropName: "propName", PropSchema: schemaInteger},
			{Container: schemaObjectInteger, Field: "OneOf", Index: 0, PropName: "propName", PropSchema: schemaInteger},
			{Container: schemaObjectString, Field: "OneOf", Index: 1, PropName: "propName", PropSchema: schemaString},
		},
	}

	if len(expected) != len(got) {
		t.Errorf("expected %d elements, got %d.\nExpected: %v\nGot.....: %v\n", len(expected), len(got), expected, got)
		return
	}

	for k, vExpected := range expected {
		vGot := got[k]
		checkObjectPropertySchemas(t, k, vExpected, vGot)
	}
}
