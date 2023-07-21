package mgc_sdk

import "github.com/getkin/kin-openapi/openapi3"

func NewObjectSchema(properties map[string]*Schema, required []string) *Schema {
	hasAdditionalProperties := false

	p := openapi3.Schemas{}
	for k, v := range properties {
		p[k] = &openapi3.SchemaRef{Value: (*openapi3.Schema)(v)}
	}

	return &Schema{
		Type:                 "object",
		AdditionalProperties: openapi3.AdditionalProperties{Has: &hasAdditionalProperties},
		Properties:           p,
		Required:             required,
	}
}

func NewStringSchema() *Schema {
	return &Schema{
		Type: "strings",
	}
}

func NewNumberSchema() *Schema {
	return &Schema{
		Type: "number",
	}
}

func NewBooleanSchema() *Schema {
	return &Schema{
		Type: "boolean",
	}
}

func NewNullSchema() *Schema {
	return &Schema{
		Type: "null",
	}
}

func NewArraySchema(item *Schema) *Schema {
	return &Schema{
		Type:  "array",
		Items: &openapi3.SchemaRef{Value: (*openapi3.Schema)(item)},
	}
}

func SetDefault(schema *Schema, value any) *Schema {
	schema.Default = value
	return schema
}

func SetDescription(schema *Schema, description string) *Schema {
	schema.Description = description
	return schema
}
