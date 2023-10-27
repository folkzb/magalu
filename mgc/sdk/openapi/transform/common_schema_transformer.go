package transform

import (
	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
)

// Common pattern that checks existing specs, if they exist then call transformSpecs(),
// otherwise process Arrays and Objects.
//
// Scalars are passed thru while Constraints() are recursively processed.
type commonSchemaTransformer[T any] struct {
	tKey                 string
	transformSpecs       func(specs []*transformSpec, value T) (T, error)
	transformArray       func(t mgcSchemaPkg.Transformer[T], schema *core.Schema, itemSchema *core.Schema, value T) (T, error)
	transformObject      func(t mgcSchemaPkg.Transformer[T], schema *core.Schema, value T) (T, error)
	transformConstraints func(t mgcSchemaPkg.Transformer[T], kind mgcSchemaPkg.ConstraintKind, schemaRefs mgcSchemaPkg.SchemaRefs, value T) (T, error)
}

func (t *commonSchemaTransformer[T]) Transform(schema *core.Schema, value T) (T, error) {
	specs := getTransformationSpecs(schema.Extensions, t.tKey)
	var err error
	if len(specs) > 0 {
		value, err = t.transformSpecs(specs, value)
		if err == nil {
			err = mgcSchemaPkg.TransformStop
		}
	}
	return value, err
}

func (t *commonSchemaTransformer[T]) Scalar(schema *core.Schema, value T) (T, error) {
	return value, nil
}

func (t *commonSchemaTransformer[T]) Array(schema *core.Schema, itemSchema *core.Schema, value T) (T, error) {
	if itemSchema == nil {
		return value, nil
	}
	return t.transformArray(t, schema, itemSchema, value)
}

func (t *commonSchemaTransformer[T]) Constraints(kind mgcSchemaPkg.ConstraintKind, schemaRefs mgcSchemaPkg.SchemaRefs, value T) (T, error) {
	return t.transformConstraints(t, kind, schemaRefs, value)
}

func (t *commonSchemaTransformer[T]) Object(schema *core.Schema, value T) (T, error) {
	return t.transformObject(t, schema, value)
}

var _ mgcSchemaPkg.Transformer[any] = (*commonSchemaTransformer[any])(nil)
