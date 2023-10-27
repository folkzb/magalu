package transform

import (
	"fmt"
	"strings"

	"github.com/stoewer/go-strcase"
	"go.uber.org/zap"
	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
)

func doTransformValue(spec *transformSpec, value any) (any, error) {
	switch spec.Type {
	case "uppercase", "upper-case", "upper":
		if s, ok := value.(string); ok {
			return strings.ToUpper(s), nil
		}
	case "lowercase", "lower-case", "lower":
		if s, ok := value.(string); ok {
			return strings.ToLower(s), nil
		}
	case "kebabcase", "kebab-case", "kebab":
		if s, ok := value.(string); ok {
			return strcase.KebabCase(s), nil
		}
	case "snakecase", "snake-case", "snake":
		if s, ok := value.(string); ok {
			return strcase.SnakeCase(s), nil
		}
	case "pascal", "pascalcase", "pascal-case", "upper-camel":
		if s, ok := value.(string); ok {
			return strcase.UpperCamelCase(s), nil
		}
	case "camel", "camelcase", "camel-case", "lower-camel":
		if s, ok := value.(string); ok {
			return strcase.LowerCamelCase(s), nil
		}
	case "regexp", "regexp-replace":
		if s, ok := value.(string); ok {
			return transformRegExp(spec.Parameters, s)
		}
	case "translate":
		return transformTranslate(spec.Parameters, value)
	}

	return value, nil
}

func doTransformsToValue(logger *zap.SugaredLogger, specs []*transformSpec, value any) (result any, err error) {
	result = value
	for _, spec := range specs {
		result, err = doTransformValue(spec, result)
		if err != nil {
			logger.Debugw("transformation attempt failed", "value", value, "type", spec.Type)
			return
		}
	}
	logger.Debugw("transformed value", "input", value, "output", result)
	return
}

// Recursively transforms the value based on the schema that may contain transformations
// If the schema doesn't contain any transformation, then the value is unchanged
func transformValue(logger *zap.SugaredLogger, schema *core.Schema, transformationKey string, value any) (any, error) {
	t := &commonSchemaTransformer[any]{
		tKey:                 transformationKey,
		transformSpecs:       func(specs []*transformSpec, value any) (any, error) { return doTransformsToValue(logger, specs, value) },
		transformArray:       transformArrayValue,
		transformObject:      transformObjectValue,
		transformConstraints: transformConstraintsValue,
	}
	return mgcSchemaPkg.Transform[any](t, schema, value)
}

func transformArrayValue(t mgcSchemaPkg.Transformer[any], schema *core.Schema, itemSchema *core.Schema, value any) (any, error) {
	valueSlice, ok := value.([]any)
	if !ok {
		return value, fmt.Errorf("expected []any, got %T %#v", value, value)
	}

	cs := utils.NewCOWSliceFunc(valueSlice, utils.IsSameValueOrPointer)
	for i, itemValue := range valueSlice {
		convertedValue, err := mgcSchemaPkg.Transform(t, itemSchema, itemValue)
		if err != nil {
			return value, err
		}
		cs.Set(i, convertedValue)
	}

	valueSlice, _ = cs.Release()
	return valueSlice, nil
}

func transformObjectValue(t mgcSchemaPkg.Transformer[any], schema *core.Schema, value any) (any, error) {
	valueMap, ok := value.(map[string]any)
	if !ok {
		return value, fmt.Errorf("expected map[string]any, got %T %#v", value, value)
	}
	cm, err := mgcSchemaPkg.TransformObjectProperties(
		schema,
		utils.NewCOWMapFunc(valueMap, utils.IsSameValueOrPointer),
		func(propName string, propSchema *core.Schema, cm *utils.COWMap[string, any],
		) (*utils.COWMap[string, any], error) {
			propValue, ok := valueMap[propName]
			if !ok {
				return cm, nil
			}

			convertedFieldValue, err := mgcSchemaPkg.Transform(t, propSchema, propValue)
			if err != nil {
				return cm, err
			}
			cm.Set(propName, convertedFieldValue)
			return cm, nil
		},
	)
	if err != nil {
		return value, err
	}

	valueMap, _ = cm.Release()
	return valueMap, nil
}

func transformConstraintsValue(t mgcSchemaPkg.Transformer[any], kind mgcSchemaPkg.ConstraintKind, schemaRefs mgcSchemaPkg.SchemaRefs, value any) (any, error) {
	// TODO: handle kind properly, see https://swagger.io/docs/specification/data-models/oneof-anyof-allof-not/
	return mgcSchemaPkg.TransformSchemasArray(t, schemaRefs, value)
}
