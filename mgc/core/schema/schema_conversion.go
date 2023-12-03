package schema

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/iancoleman/orderedmap"
	"github.com/invopop/jsonschema"
	"golang.org/x/exp/constraints"
)

func ToCoreSchema(s *jsonschema.Schema) (schema *Schema, err error) {
	if s == nil {
		return nil, fmt.Errorf("invalid jsonschema.Schema passed to 'toCoreSchema' function")
	}

	rootSchema := s
	if s.Ref != "" {
		rootDef, ok := lookupDefByPath(s.Definitions, getRefPath(s.Ref))
		if !ok {
			return nil, fmt.Errorf("unable to resolve reference %s when generating schema via 'toCoreSchema'", s.Ref)
		}
		rootSchema = rootDef
	}

	oapiSchema := convertJsonSchemaToOpenAPISchema(rootSchema)
	refCache := map[string]*openapi3.Schema{}
	err = resolveRefs(oapiSchema, s.Definitions, refCache)
	if err != nil {
		return nil, err
	}

	initializeExtensions(oapiSchema)
	schema = (*Schema)(oapiSchema)
	return SimplifySchema(schema)
}

func isJsonSchemaSchemaNullable(input *jsonschema.Schema) bool {
	for _, sub := range input.OneOf {
		if sub != nil && sub.Type == "null" {
			return true
		}
	}
	return false
}

func convertJsonSchemaNumberToOpenAPIPointer[T constraints.Integer | constraints.Float](v int) (r *T) {
	if v == 0 {
		return nil
	}
	r = new(T)
	*r = T(v)
	return
}

func addExtensions(output *openapi3.Schema, name string, value any) {
	if output.Extensions == nil {
		output.Extensions = map[string]interface{}{}
	}
	output.Extensions[name] = value
}

func convertJsonSchemaToOpenAPISchema(input *jsonschema.Schema) (output *openapi3.Schema) {
	if input == nil {
		return nil
	}

	// We used to MarshalJSON() from jsonschema.Schema and UnmarshalJSON() into openapi3.Schema, but
	// jsonschema's MarshalJSON() will return "true" for empty schema and this is not handled by openapi3.Schema's UnmarshalJSON()
	// Then do it manually.

	if input == jsonschema.TrueSchema {
		return (*openapi3.Schema)(NewAnySchema())
	}

	additionalProperties := openapi3.AdditionalProperties{}
	if input.AdditionalProperties != nil && input.AdditionalProperties != jsonschema.FalseSchema {
		has := true
		additionalProperties.Has = &has
		additionalProperties.Schema = convertJsonSchemaToOpenAPISchemaRef(input.AdditionalProperties)
	}
	if len(input.PatternProperties) > 0 && additionalProperties.Has == nil {
		has := true
		additionalProperties.Has = &has
		additionalProperties.Schema = &openapi3.SchemaRef{
			Value: openapi3.NewAnyOfSchema(convertJsonSchemaToOpenAPISchemaMapToSlice(input.PatternProperties)...),
		}
	}

	output = &openapi3.Schema{
		OneOf:        convertJsonSchemaToOpenAPISchemaSlice(input.OneOf),
		AnyOf:        convertJsonSchemaToOpenAPISchemaSlice(input.AnyOf),
		AllOf:        convertJsonSchemaToOpenAPISchemaSlice(input.AllOf),
		Not:          convertJsonSchemaToOpenAPISchemaRef(input.Not),
		Type:         input.Type,
		Title:        input.Title,
		Format:       input.Format,
		Description:  input.Description,
		Enum:         input.Enum,
		Default:      input.Default,
		UniqueItems:  input.UniqueItems,
		ExclusiveMin: input.ExclusiveMinimum,
		ExclusiveMax: input.ExclusiveMaximum,
		Nullable:     isJsonSchemaSchemaNullable(input),
		ReadOnly:     input.ReadOnly,
		WriteOnly:    input.WriteOnly,
		// Does not exist: AllowEmptyValue:      input.AllowEmptyValue,
		Deprecated:           input.Deprecated,
		Min:                  convertJsonSchemaNumberToOpenAPIPointer[float64](input.Minimum),
		Max:                  convertJsonSchemaNumberToOpenAPIPointer[float64](input.Maximum),
		MultipleOf:           convertJsonSchemaNumberToOpenAPIPointer[float64](input.MultipleOf),
		MinLength:            uint64(input.MinLength),
		MaxLength:            convertJsonSchemaNumberToOpenAPIPointer[uint64](input.MaxLength),
		Pattern:              input.Pattern,
		MinItems:             uint64(input.MinItems),
		MaxItems:             convertJsonSchemaNumberToOpenAPIPointer[uint64](input.MaxItems),
		Items:                convertJsonSchemaToOpenAPISchemaRef(input.Items),
		Required:             input.Required,
		Properties:           convertJsonSchemaToOpenAPISchemaMap(input.Properties),
		MinProps:             uint64(input.MinProperties),
		MaxProps:             convertJsonSchemaNumberToOpenAPIPointer[uint64](input.MaxProperties),
		AdditionalProperties: additionalProperties,
		// Does not exist: Discriminator:        input.Discriminator,
	}

	if len(input.Examples) > 0 {
		output.Example = input.Examples[0]
	}

	if input.ContentMediaType != "" {
		addExtensions(output, "x-contentMediaType", input.ContentMediaType)
	}
	if input.ContentEncoding != "" {
		addExtensions(output, "x-contentEncoding", input.ContentEncoding)
	}
	if input.ContentSchema != nil {
		addExtensions(output, "x-contentSchema", input.ContentSchema)
	}

	if reflect.DeepEqual(&Schema{}, output) {
		return (*openapi3.Schema)(NewAnySchema())
	}

	return
}

func convertJsonSchemaToOpenAPISchemaRef(input *jsonschema.Schema) (outpyt *openapi3.SchemaRef) {
	if input == nil {
		return nil
	}
	s := convertJsonSchemaToOpenAPISchema(input)
	if s == nil {
		return nil
	}
	return &openapi3.SchemaRef{Value: s}
}

func convertJsonSchemaToOpenAPISchemaSlice(input []*jsonschema.Schema) (output []*openapi3.SchemaRef) {
	if len(input) == 0 {
		return nil
	}
	output = make([]*openapi3.SchemaRef, len(input))
	for i, value := range input {
		output[i] = convertJsonSchemaToOpenAPISchemaRef(value)
	}
	return
}

func convertJsonSchemaToOpenAPISchemaMap(input *orderedmap.OrderedMap) (output map[string]*openapi3.SchemaRef) {
	if input == nil {
		return nil
	}
	values := input.Values()
	if len(values) == 0 {
		return nil
	}
	output = make(map[string]*openapi3.SchemaRef, len(values))
	for key, value := range values {
		output[key] = convertJsonSchemaToOpenAPISchemaRef(value.(*jsonschema.Schema))
	}
	return
}

func convertJsonSchemaToOpenAPISchemaMapToSlice(input map[string]*jsonschema.Schema) (output []*openapi3.Schema) {
	if len(input) == 0 {
		return nil
	}
	output = make([]*openapi3.Schema, 0, len(input))
	for _, value := range input {
		output = append(output, convertJsonSchemaToOpenAPISchema(value))
	}
	return

}

func getRefPath(ref string) []string {
	ref = strings.TrimPrefix(ref, "#/$defs/")
	return strings.Split(ref, "/")
}

func lookupDefByPath(defs jsonschema.Definitions, path []string) (*jsonschema.Schema, bool) {
	pathLength := len(path)
	if pathLength == 0 {
		return nil, false
	}

	def, ok := defs[path[0]]
	if !ok {
		return nil, false
	}

	if pathLength > 1 {
		if def.Definitions != nil {
			return lookupDefByPath(def.Definitions, path[1:])
		} else {
			return nil, false
		}
	} else {
		return def, true
	}
}

func initializeExtensions(s *openapi3.Schema) {
	if s.Extensions == nil {
		s.Extensions = make(map[string]any)
	}
	_, _ = visitAllSubRefs(s, func(ref *openapi3.SchemaRef) error {
		if ref.Value.Extensions == nil {
			ref.Value.Extensions = make(map[string]any)
		}

		return nil
	})
}

type subRefVisitor func(ref *openapi3.SchemaRef) error

func visitAllSubRefs(s *openapi3.Schema, visitor subRefVisitor) (bool, error) {
	if s == nil {
		return true, nil
	}

	if s.AdditionalProperties.Schema != nil {
		if err := visitor(s.AdditionalProperties.Schema); err != nil {
			return false, err
		}
		if shouldContinue, err := visitAllSubRefs(s.AdditionalProperties.Schema.Value, visitor); !shouldContinue {
			return false, err
		}
	}

	if s.OneOf != nil {
		for _, s := range s.OneOf {
			if err := visitor(s); err != nil {
				return false, err
			}
			if shouldContinue, err := visitAllSubRefs(s.Value, visitor); !shouldContinue {
				return false, err
			}
		}
	}
	if s.AnyOf != nil {
		for _, s := range s.AnyOf {
			if err := visitor(s); err != nil {
				return false, err
			}
			if shouldContinue, err := visitAllSubRefs(s.Value, visitor); !shouldContinue {
				return false, err
			}
		}
	}
	if s.AllOf != nil {
		for _, s := range s.AllOf {
			if err := visitor(s); err != nil {
				return false, err
			}
			if shouldContinue, err := visitAllSubRefs(s.Value, visitor); !shouldContinue {
				return false, err
			}
		}
	}
	if s.Properties != nil {
		for _, s := range s.Properties {
			if err := visitor(s); err != nil {
				return false, err
			}
			if shouldContinue, err := visitAllSubRefs(s.Value, visitor); !shouldContinue {
				return false, err
			}
		}
	}
	if s.Not != nil {
		if err := visitor(s.Not); err != nil {
			return false, err
		}
		if shouldContinue, err := visitAllSubRefs(s.Not.Value, visitor); !shouldContinue {
			return false, err
		}
	}
	if s.Items != nil {
		if err := visitor(s.Items); err != nil {
			return false, err
		}
		if shouldContinue, err := visitAllSubRefs(s.Items.Value, visitor); !shouldContinue {
			return false, err
		}
	}
	return true, nil
}

func resolveRefs(s *openapi3.Schema, defs jsonschema.Definitions, cache map[string]*openapi3.Schema) error {
	_, err := visitAllSubRefs(s, func(ref *openapi3.SchemaRef) error {
		if ref == nil || ref.Value != nil {
			return nil
		}

		if ref.Ref == "" {
			return fmt.Errorf("schema with empty reference passed to 'toCoreSchema'")
		}

		if def, ok := findRef(getRefPath(ref.Ref), defs, cache); ok {
			ref.Value = def
			return resolveRefs(def, defs, cache)
		}

		return fmt.Errorf("unable to resolve %s reference when generating schema via 'toCoreSchema' function", ref.Ref)
	})
	return err
}

func findRef(refPath []string, defs jsonschema.Definitions, cache map[string]*openapi3.Schema) (*openapi3.Schema, bool) {
	refPathLength := len(refPath)
	if refPathLength == 0 {
		return nil, false
	}

	refName := strings.Join(refPath, "/")
	if def, ok := cache[refName]; ok {
		return def, true
	}

	def, ok := lookupDefByPath(defs, refPath)
	if !ok {
		return nil, false
	}

	oapiDef := convertJsonSchemaToOpenAPISchema(def)
	err := resolveRefs(oapiDef, defs, cache)
	if err != nil {
		return nil, false
	}

	cache[refName] = oapiDef
	return oapiDef, true
}
