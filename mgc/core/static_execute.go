package core

type StaticExecute struct {
	name        string
	version     string
	description string
	parameters  *Schema
	config      *Schema
	execute     func(parameters map[string]Value, configs map[string]Value) (result Value, err error)
}

func NewStaticExecute(name string, version string, description string, parameters *Schema, config *Schema, execute func(parameters map[string]Value, configs map[string]Value) (result Value, err error)) *StaticExecute {
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

func (o *StaticExecute) ParametersSchema() *Schema {
	return o.parameters
}

func (o *StaticExecute) ConfigsSchema() *Schema {
	return o.config
}

func (o *StaticExecute) Execute(parameters map[string]Value, configs map[string]Value) (result Value, err error) {
	return o.execute(parameters, configs)
}

var _ Executor = (*StaticExecute)(nil)

// END: Executor interface
