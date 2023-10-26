package core

import (
	"context"
	"fmt"
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/invopop/jsonschema"
	"magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
)

// Given a base spec and types to be reflected, populate with the schemas
func ReflectExecutorSpecSchemas[ParamsT any, ConfigsT any, ResultT any](baseSpec ExecutorSpec) (spec ExecutorSpec, err error) {
	spec = baseSpec

	spec.ParametersSchema, err = schemaFromType[ParamsT]()
	if err != nil {
		err = &ChainedError{Name: "ParamsT", Err: err}
		return
	}
	spec.ConfigsSchema, err = schemaFromType[ConfigsT]()
	if err != nil {
		err = &ChainedError{Name: "ConfigsT", Err: err}
		return
	}
	spec.ResultSchema, err = schemaFromType[ResultT]()
	if err != nil {
		err = &ChainedError{Name: "ResultT", Err: err}
		return
	}

	return spec, nil
}

func ReflectExecutorSpecFn[ParamsT any, ConfigsT any, ResultT any](
	typedExecute func(context context.Context, params ParamsT, configs ConfigsT) (result ResultT, err error),
) ExecutorSpecFn {
	return func(executor Executor, ctx context.Context, parameters Parameters, configs Configs) (Result, error) {
		typedParams, err := utils.DecodeNewValue[ParamsT](parameters)
		if err != nil {
			return nil, &ChainedError{Name: "parameters", Err: fmt.Errorf("decoding error. Did you forget to set 'json' struct flags for struct %T?: %w", typedParams, err)}
		}

		typedConfigs, err := utils.DecodeNewValue[ConfigsT](configs)
		if err != nil {
			return nil, &ChainedError{Name: "configs", Err: fmt.Errorf("decoding error. Did you forget to set 'json' struct flags for struct %T?: %w", typedConfigs, err)}
		}

		typedResult, err := typedExecute(ctx, *typedParams, *typedConfigs)
		if err != nil {
			return nil, err
		}

		value, err := utils.SimplifyAny(typedResult)
		if err != nil {
			return nil, &ChainedError{Name: "result", Err: fmt.Errorf("error simplifying %T: %w", typedResult, err)}
		}

		source := ResultSource{
			Executor:   executor,
			Context:    ctx,
			Parameters: parameters,
			Configs:    configs,
		}
		return NewSimpleResult(source, executor.ResultSchema(), value), nil
	}
}

func ReflectExecutorSpec[ParamsT any, ConfigsT any, ResultT any](
	baseSpec ExecutorSpec,
	typedExecute func(context context.Context, params ParamsT, configs ConfigsT) (result ResultT, err error),
) (spec ExecutorSpec, err error) {
	spec, err = ReflectExecutorSpecSchemas[ParamsT, ConfigsT, ResultT](baseSpec)
	if err != nil {
		return
	}

	spec.Execute = ReflectExecutorSpecFn[ParamsT, ConfigsT, ResultT](typedExecute)
	return
}

func newAnySchema() *Schema {
	s := &openapi3.Schema{
		Nullable: true,
		AnyOf: openapi3.SchemaRefs{
			&openapi3.SchemaRef{Value: &openapi3.Schema{Type: "null", Nullable: true}},
			&openapi3.SchemaRef{Value: openapi3.NewBoolSchema()},
			&openapi3.SchemaRef{Value: openapi3.NewStringSchema()},
			&openapi3.SchemaRef{Value: openapi3.NewFloat64Schema()},
			&openapi3.SchemaRef{Value: openapi3.NewIntegerSchema()},
			&openapi3.SchemaRef{Value: openapi3.NewArraySchema()},
			&openapi3.SchemaRef{Value: openapi3.NewObjectSchema().WithAnyAdditionalProperties()},
		},
	}

	return (*Schema)(s)
}

func schemaFromType[T any]() (*Schema, error) {
	t := new(T)
	tp := reflect.TypeOf(t).Elem()
	kind := tp.Kind()
	if tp.Name() == "" && kind == reflect.Interface {
		return newAnySchema(), nil
	}

	s, err := schema.ToCoreSchema(schemaReflector.Reflect(t))
	if err != nil {
		return nil, fmt.Errorf("unable to create JSON Schema for type '%T': %w", t, err)
	}

	isArray := kind == reflect.Array || kind == reflect.Slice

	// schemaReflector seems to lose the fact that it's an array, so we bring that back
	if isArray && s.Type == "object" {
		arrSchema := schema.NewArraySchema(s)
		s = arrSchema
	}

	return s, nil
}

// Go Parameter and Config structs
// Note: we use both 'jsonschema' and 'mapstructure' for this helper. Be careful
// when using struct tags in your Params and Configs structs, as the tags from those
// libraries can't be out of sync when it comes to field names/json names
// See:
// - https://pkg.go.dev/github.com/invopop/jsonschema
// - https://pkg.go.dev/github.com/mitchellh/mapstructure
func NewReflectedSimpleExecutor[ParamsT any, ConfigsT any, ResultT any](
	baseSpec ExecutorSpec,
	execute func(context context.Context, params ParamsT, configs ConfigsT) (result ResultT, err error),
) *SimpleExecutor {
	execSpec, err := ReflectExecutorSpec[ParamsT, ConfigsT, ResultT](baseSpec, execute)
	if err != nil {
		logger().Fatalw("cannot reflect static executor", "err", err)
	}
	return NewSimpleExecutor(execSpec)
}

// Version that generates the schema, but uses baseSpec.Execute as is
func NewReflectedSimpleExecutorSchemas[ParamsT any, ConfigsT any, ResultT any](
	baseSpec ExecutorSpec,
) *SimpleExecutor {
	execSpec, err := ReflectExecutorSpecSchemas[ParamsT, ConfigsT, ResultT](baseSpec)
	if err != nil {
		logger().Fatalw("cannot reflect static executor", "err", err)
	}
	return NewSimpleExecutor(execSpec)
}

// Go Parameter and Config structs
// Note: we use both 'jsonschema' and 'mapstructure' for this helper. Be careful
// when using struct tags in your Params and Configs structs, as the tags from those
// libraries can't be out of sync when it comes to field names/json names
// See:
// - https://pkg.go.dev/github.com/invopop/jsonschema
// - https://pkg.go.dev/github.com/mitchellh/mapstructure
func NewStaticExecute[ParamsT any, ConfigsT any, ResultT any](
	spec DescriptorSpec,
	execute func(context context.Context, params ParamsT, configs ConfigsT) (result ResultT, err error),
) *SimpleExecutor {
	return NewReflectedSimpleExecutor(ExecutorSpec{DescriptorSpec: spec}, execute)
}

// No parameters or configs
func NewStaticExecuteSimple[ResultT any](
	spec DescriptorSpec,
	execute func(context context.Context) (result ResultT, err error),
) *SimpleExecutor {
	return NewReflectedSimpleExecutor(
		ExecutorSpec{DescriptorSpec: spec},
		func(context context.Context, _ struct{}, _ struct{}) (result ResultT, err error) {
			return execute(context)
		},
	)
}

var schemaReflector *jsonschema.Reflector

func init() {
	schemaReflector = &jsonschema.Reflector{
		DoNotReference: false,
	}
}
