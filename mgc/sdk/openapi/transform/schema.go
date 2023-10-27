package transform

import (
	"fmt"
	"reflect"

	"go.uber.org/zap"
	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
)

func doTransformSchema(spec *transformSpec, value *mgcSchemaPkg.COWSchema) (*mgcSchemaPkg.COWSchema, error) {
	if spec.Schema != nil {
		value.Replace((*mgcSchemaPkg.Schema)(spec.Schema))
		return value, nil
	}
	switch spec.Type {
	default:
		return value, nil
	case "translate":
		return transformTranslateSchema(spec.Parameters, value)
	}
}

func doTransformsToSchema(logger *zap.SugaredLogger, specs []*transformSpec, value *mgcSchemaPkg.COWSchema) (result *mgcSchemaPkg.COWSchema, err error) {
	result = value
	for _, spec := range specs {
		result, err = doTransformSchema(spec, result)
		if err != nil {
			logger.Debugw("transformation attempt failed", "value", value, "type", spec.Type)
			return
		}
	}
	logger.Debugw("transformed schema", "input", value.Peek(), "output", result.Peek())
	return
}

func transformSchema(logger *zap.SugaredLogger, schema *core.Schema, transformationKey string, value *core.Schema) (*core.Schema, error) {
	t := &commonSchemaTransformer[*mgcSchemaPkg.COWSchema]{
		tKey: transformationKey,
		transformSpecs: func(specs []*transformSpec, value *mgcSchemaPkg.COWSchema) (*mgcSchemaPkg.COWSchema, error) {
			return doTransformsToSchema(logger, specs, value)
		},
		transformArray:       transformArraySchema,
		transformObject:      transformObjectSchema,
		transformConstraints: transformConstraintsSchema,
	}
	cowSchema := mgcSchemaPkg.NewCOWSchema(value)
	cowSchema, err := mgcSchemaPkg.Transform[*mgcSchemaPkg.COWSchema](t, schema, cowSchema)
	if err != nil {
		return value, err
	}
	return cowSchema.Peek(), nil
}

func transformArraySchema(t mgcSchemaPkg.Transformer[*mgcSchemaPkg.COWSchema], schema *core.Schema, itemSchema *core.Schema, value *mgcSchemaPkg.COWSchema) (*mgcSchemaPkg.COWSchema, error) {
	itemsCow := value.ItemsCOW().ValueCOW()
	_, err := mgcSchemaPkg.Transform(t, itemSchema, itemsCow)
	if err != nil {
		return nil, err
	}

	return value, nil
}

func transformObjectSchema(t mgcSchemaPkg.Transformer[*mgcSchemaPkg.COWSchema], schema *core.Schema, value *mgcSchemaPkg.COWSchema) (*mgcSchemaPkg.COWSchema, error) {
	_, err := mgcSchemaPkg.TransformObjectProperties(
		schema,
		value.PropertiesCOW(),
		func(propName string, propSchema *core.Schema, propertiesCow *utils.COWMapOfCOW[string, *mgcSchemaPkg.SchemaRef, *mgcSchemaPkg.COWSchemaRef],
		) (*utils.COWMapOfCOW[string, *mgcSchemaPkg.SchemaRef, *mgcSchemaPkg.COWSchemaRef], error) {
			propSchemaCow, ok := propertiesCow.GetCOW(propName)
			if !ok {
				return nil, fmt.Errorf("schema missing property %q", propName) // this should never happen
			}

			_, err := mgcSchemaPkg.Transform(t, propSchema, propSchemaCow.ValueCOW())
			if err != nil {
				return nil, err
			}
			return propertiesCow, nil
		},
	)
	if err != nil {
		return value, err
	}
	return value, nil
}

func transformConstraintsSchema(t mgcSchemaPkg.Transformer[*mgcSchemaPkg.COWSchema], kind mgcSchemaPkg.ConstraintKind, schemaRefs mgcSchemaPkg.SchemaRefs, value *mgcSchemaPkg.COWSchema) (result *mgcSchemaPkg.COWSchema, err error) {
	result = value

	if kind == mgcSchemaPkg.ConstraintNot {
		_, err = mgcSchemaPkg.Transform(t, (*mgcSchemaPkg.Schema)(schemaRefs[0].Value), value.NotCOW().ValueCOW())
		return
	}

	var constraintCow *utils.COWSliceOfCOW[*mgcSchemaPkg.SchemaRef, *mgcSchemaPkg.COWSchemaRef]
	switch kind {
	case mgcSchemaPkg.ConstraintAllOf:
		constraintCow = value.AllOfCOW()
	case mgcSchemaPkg.ConstraintAnyOf:
		constraintCow = value.AnyOfCOW()
	case mgcSchemaPkg.ConstraintOneOf:
		constraintCow = value.OneOfCOW()
	default:
		return value, fmt.Errorf("unknown constraint kind: %q", kind)
	}

	constraintCow.ForEachCOW(func(i int, cowRef *mgcSchemaPkg.COWSchemaRef) (run bool) {
		itemSchema := cowRef.Peek()
		if itemSchema == nil {
			return true
		}

		_, err = mgcSchemaPkg.Transform(t, (*mgcSchemaPkg.Schema)(itemSchema.Value), cowRef.ValueCOW())
		return err == nil
	})

	return
}

func reverseTranslate(spec *transformTranslateSpec, value any) (any, error) {
	for _, item := range spec.Translations {
		if reflect.DeepEqual(item.To, value) {
			return item.From, nil
		}
	}
	if spec.AllowMissing {
		return value, nil
	}
	return value, fmt.Errorf("translation not found: %#v", value)
}

func transformTranslateSchema(params map[string]any, schema *mgcSchemaPkg.COWSchema) (result *mgcSchemaPkg.COWSchema, err error) {
	if schema.Default() == nil && len(schema.Enum()) == 0 {
		return schema, nil
	}

	spec, err := utils.DecodeNewValue[transformTranslateSpec](params)
	if err != nil {
		return schema, fmt.Errorf("invalid translation parameters: %w", err)
	}
	if len(spec.Translations) == 0 {
		return schema, fmt.Errorf("invalid translation parameters: missing translations")
	}

	result = schema

	if schema.Default() != nil {
		var schemaDefault any
		schemaDefault, err = reverseTranslate(spec, schema.Default())
		if err != nil {
			return
		}
		schema.SetDefault(schemaDefault)
	}

	enumCow := schema.EnumCOW()
	enumCow.ForEach(func(i int, value any) (run bool) {
		var translatedEnum any
		translatedEnum, err = reverseTranslate(spec, value)
		if err != nil {
			return false
		}
		enumCow.Set(i, translatedEnum)
		return true
	})

	return
}
