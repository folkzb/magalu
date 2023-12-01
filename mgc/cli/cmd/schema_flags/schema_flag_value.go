package schema_flags

import (
	flag "github.com/spf13/pflag"
	"magalu.cloud/core"
)

type SchemaFlagValue interface {
	flag.Value

	Desc() SchemaFlagValueDesc

	RawDefaultValue() string
	RawNoOptDefVal() string
	Usage() string
	Parse() (any, error)

	Changed() bool // if the value was explicitly set
}

func newSchemaFlagValue(desc SchemaFlagValueDesc) SchemaFlagValue {
	switch desc.Schema.Type {
	case "array":
		return newSchemaFlagValueArray(desc)
	case "boolean":
		return newSchemaFlagValueBool(desc)
	case "string":
		return newSchemaFlagValueString(desc)
	case "object":
		return newSchemaFlagValueObject(desc)
	default:
		return newSchemaFlagValueCommon(desc)
	}
}

func newFlag(value SchemaFlagValue) *flag.Flag {
	return &flag.Flag{
		Name:        string(value.Desc().FlagName),
		DefValue:    value.RawDefaultValue(),
		NoOptDefVal: value.RawNoOptDefVal(),
		Usage:       value.Usage(),
		Value:       value,
	}
}

func NewSchemaFlag(
	container *core.Schema, // the object that contains this property
	propName string, // name inside ParametersSchema()/ConfigsSchema() Properties
	flagName flag.NormalizedName, // public/user-visible name after normalization
	isRequired bool,
	isConfig bool,
) *flag.Flag {
	schema := (*core.Schema)(container.Properties[propName].Value)
	return newFlag(newSchemaFlagValue(SchemaFlagValueDesc{
		container,
		schema,
		propName,
		flagName,
		isRequired,
		isConfig,
	}))
}
