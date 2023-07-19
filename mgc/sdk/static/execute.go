package static

import "sdk"

type StaticExecute struct {
	name        string
	version     string
	description string
	parameters  *sdk.Schema
	config      *sdk.Schema
	execute     func(parameters map[string]sdk.Value, configs map[string]sdk.Value) (result sdk.Value, err error)
}

func NewStaticExecute(name string, version string, description string, parameters *sdk.Schema, config *sdk.Schema, execute func(parameters map[string]sdk.Value, configs map[string]sdk.Value) (result sdk.Value, err error)) *StaticExecute {
	return &StaticExecute{name, version, description, parameters, config, execute}
}

// BEGIN: Descriptor interface:

func (o *StaticExecute) Name() string {
	return o.name
}

func (o *StaticExecute) Version() string {
	return o.version
}

func (o *StaticExecute) Description() string {
	return o.description
}

// END: Descriptor interface

// BEGIN: Executor interface:

func (o *StaticExecute) ParametersSchema() *sdk.Schema {
	return o.parameters
}

func (o *StaticExecute) ConfigsSchema() *sdk.Schema {
	return o.config
}

func (o *StaticExecute) Execute(parameters map[string]sdk.Value, configs map[string]sdk.Value) (result sdk.Value, err error) {
	return o.execute(parameters, configs)
}

var _ sdk.Executor = (*StaticExecute)(nil)

// END: Executor interface
