package mgc_sdk

import "github.com/getkin/kin-openapi/openapi3"

// NOTE: TODO: should we duplicate this, or find a more generic package?
type Schema openapi3.Schema

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
type Value any

// Type comes from the Schema
type Example Value

type Parameter interface {
	Name() string
	Description() string
	Required() bool
	Schema() *Schema
	Examples() []Example
}

// Config are similar to Parameters, but for less variant parts,
// usually saved in configuration files
// So far it's the same, but let's be future-proof
type Config Parameter

type Executor interface {
	Descriptor
	Parameters() map[string]Parameter
	Configs() map[string]Config
	Execute(parameters map[string]Value, configs map[string]Value) (result Value, err error)
}
