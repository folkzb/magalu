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

type StaticExecute struct {
	SimpleDescriptor
	parameters *Schema
	config     *Schema
	result     *Schema
	links      map[string]Linker
	related    map[string]Executor
	execute    func(ctx context.Context, parameters Parameters, configs Configs) (value Value, err error)
}

// Raw Parameter and Config JSON Schemas
func NewRawStaticExecute(spec DescriptorSpec, parameters *Schema, config *Schema, result *Schema, links map[string]Linker, related map[string]Executor, execute func(context context.Context, parameters Parameters, configs Configs) (value Value, err error)) *StaticExecute {
	return &StaticExecute{SimpleDescriptor{spec}, parameters, config, result, links, related, execute}
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
func NewStaticExecuteWithLinksAndRelated[ParamsT any, ConfigsT any, ResultT any](
	spec DescriptorSpec,
	links map[string]Linker,
	related map[string]Executor,
	execute func(context context.Context, params ParamsT, configs ConfigsT) (result ResultT, err error),
) *StaticExecute {
	ps, err := schemaFromType[ParamsT]()
	if err != nil {
		logger().Fatal(err)
	}
	cs, err := schemaFromType[ConfigsT]()
	if err != nil {
		logger().Fatal(err)
	}
	rs, err := schemaFromType[ResultT]()
	if err != nil {
		logger().Fatal(err)
	}

	return NewRawStaticExecute(
		spec,
		ps,
		cs,
		rs,
		links,
		related,
		func(ctx context.Context, parameters Parameters, configs Configs) (Value, error) {
			paramsStruct, err := utils.DecodeNewValue[ParamsT](parameters)
			if err != nil {
				return nil, fmt.Errorf("error when decoding parameters. Did you forget to set 'json' struct flags for struct %T?: %w", paramsStruct, err)
			}

			configsStruct, err := utils.DecodeNewValue[ConfigsT](configs)
			if err != nil {
				return nil, fmt.Errorf("error when decoding configs. Did you forget to set 'json' struct flags for struct %T?: %w", paramsStruct, err)
			}

			value, err := execute(ctx, *paramsStruct, *configsStruct)
			if err != nil {
				return nil, err
			}

			return utils.SimplifyAny(value)
		},
	)
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
) *StaticExecute {
	return NewStaticExecuteWithLinksAndRelated(spec, nil, nil, execute)
}

// No parameters or configs
func NewStaticExecuteSimpleWithLinksAndRelated[ResultT any](
	spec DescriptorSpec,
	links map[string]Linker,
	related map[string]Executor,
	execute func(ctx context.Context) (result ResultT, err error),
) *StaticExecute {
	return NewStaticExecuteWithLinksAndRelated(
		spec,
		links,
		nil,
		func(ctx context.Context, _, _ struct{}) (ResultT, error) {
			return execute(ctx)
		},
	)
}

// No parameters or configs
func NewStaticExecuteSimple[ResultT any](
	spec DescriptorSpec,
	execute func(context context.Context) (result ResultT, err error),
) *StaticExecute {
	return NewStaticExecuteSimpleWithLinksAndRelated(spec, nil, nil, execute)
}

func (o *StaticExecute) ParametersSchema() *Schema {
	return o.parameters
}

func (o *StaticExecute) ConfigsSchema() *Schema {
	return o.config
}

func (o *StaticExecute) ResultSchema() *Schema {
	return o.result
}

func (o *StaticExecute) Execute(context context.Context, parameters Parameters, configs Configs) (result Result, err error) {
	value, err := o.execute(context, parameters, configs)
	if err != nil {
		return nil, err
	}
	source := ResultSource{
		Executor:   o,
		Context:    context,
		Parameters: parameters,
		Configs:    configs,
	}
	return NewSimpleResult(source, o.result, value), nil
}

func (o *StaticExecute) Links() map[string]Linker {
	return o.links
}

func (o *StaticExecute) Related() map[string]Executor {
	return o.related
}

// implemented by embedded SimpleDescriptor
var _ Executor = (*StaticExecute)(nil)

var schemaReflector *jsonschema.Reflector

func init() {
	schemaReflector = &jsonschema.Reflector{
		DoNotReference: false,
	}
}
