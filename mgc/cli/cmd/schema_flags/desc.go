package schema_flags

import (
	"encoding/json"
	"fmt"

	flag "github.com/spf13/pflag"
	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
)

type SchemaFlagValueDesc struct {
	Container *core.Schema
	Schema    *core.Schema
	PropName  string              // name inside ParametersSchema()/ConfigsSchema() Properties
	FlagName  flag.NormalizedName // public/user-visible name after normalization

	IsRequired bool
	IsConfig   bool
}

func getFlagType(schema *core.Schema) string {
	if len(schema.Enum) > 0 {
		return "enum"
	}

	if schema.Format != "" {
		return schema.Format
	}

	if schema.Type == "" {
		if (mgcSchemaPkg.CheckSimilarJsonSchemas(schema, &mgcSchemaPkg.Schema{}) || // this is like a bug in the schema, but config set takes it
			mgcSchemaPkg.CheckSimilarJsonSchemas(schema, mgcSchemaPkg.NewAnySchema())) {
			return "anyValue"
		}
		if len(schema.AnyOf) > 0 {
			return "anyOf"
		}
		if len(schema.OneOf) > 0 {
			return "oneOf"
		}
		return "anyValue"
	}

	return schema.Type
}

// To be used in flag.Value.Type().
//
// Cobra will show this one in their usage and decide some behavior
// based on it, there are special cases for "bool", "string"...
func (d *SchemaFlagValueDesc) FlagType() string {
	return getFlagType(d.Schema)
}

// To be used in flag.DefValue.
func (d *SchemaFlagValueDesc) RawDefaultValue() string {
	if d.Schema.Default == nil {
		return ""
	}

	data, err := json.Marshal(d.Schema.Default)
	if err != nil {
		logger().Warnw(
			"could not convert flag default value to string",
			"defaultValue", d.Schema.Default,
			"flagName", d.FlagName,
			"propName", d.PropName,
			"error", err,
		)
		return ""
	}

	return string(data)
}

func (d *SchemaFlagValueDesc) Usage() (usage string) {
	usage = d.Schema.Title

	if d.Schema.Description != "" {
		if usage != "" {
			usage += ": "
		}
		usage += d.Schema.Description
	}

	constraints := schemaJSONRepAndConstraints(d.Schema, false)
	if constraints != "" {
		if usage != "" {
			usage += " "
		}
		usage += fmt.Sprintf("(%s)", constraints)
	}

	return usage
}
