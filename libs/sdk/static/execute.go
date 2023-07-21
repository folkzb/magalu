package mgc_static

import "mgc_sdk"

type StaticExecute struct {
	name        string
	version     string
	description string
	parameters  *mgc_sdk.Schema
	config      *mgc_sdk.Schema
	execute     func(parameters map[string]mgc_sdk.Value, configs map[string]mgc_sdk.Value) (result mgc_sdk.Value, err error)
}

func NewStaticExecute(name string, version string, description string, parameters *mgc_sdk.Schema, config *mgc_sdk.Schema, execute func(parameters map[string]mgc_sdk.Value, configs map[string]mgc_sdk.Value) (result mgc_sdk.Value, err error)) *StaticExecute {
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

func (o *StaticExecute) ParametersSchema() *mgc_sdk.Schema {
	return o.parameters
}

func (o *StaticExecute) ConfigsSchema() *mgc_sdk.Schema {
	return o.config
}

func (o *StaticExecute) Execute(parameters map[string]mgc_sdk.Value, configs map[string]mgc_sdk.Value) (result mgc_sdk.Value, err error) {
	return o.execute(parameters, configs)
}

var _ mgc_sdk.Executor = (*StaticExecute)(nil)

// END: Executor interface
