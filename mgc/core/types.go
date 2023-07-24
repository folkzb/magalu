package core

import "github.com/getkin/kin-openapi/openapi3"

// NOTE: TODO: should we duplicate this, or find a more generic package?
type Schema openapi3.Schema

func (s *Schema) VisitJSON(value any, opts ...openapi3.SchemaValidationOption) error {
	return (*openapi3.Schema)(s).VisitJSON(value, opts...)
}

// General interface that describes both Executor and Grouper
type Descriptor interface {
	Name() string
	Version() string
	Description() string
}

type DescriptorVisitor func(child Descriptor) (run bool, err error)

type Grouper interface {
	Descriptor
	VisitChildren(visitor DescriptorVisitor) (finished bool, err error)
	GetChildByName(name string) (child Descriptor, err error)
}

// Type comes from the Schema
type Value = any

// Type comes from the Schema
type Example = Value

type Executor interface {
	Descriptor
	ParametersSchema() *Schema
	ConfigsSchema() *Schema
	Execute(parameters map[string]Value, configs map[string]Value) (result Value, err error)
}
