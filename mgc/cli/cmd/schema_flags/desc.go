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

const (
	FlagTypeFile      = "file"
	FlagTypeDirectory = "directory"
)

func getFlagType(schema *core.Schema) string {
	if len(schema.Enum) > 0 {
		return "enum"
	}

	if schema.Format != "" {
		return schema.Format
	}

	if mt := schema.Extensions["x-contentMediaType"]; mt != nil {
		if mt == "inode/directory" {
			return FlagTypeDirectory
		}
		return FlagTypeFile
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

func (d SchemaFlagValueDesc) Description() (description string) {
	description = d.Schema.Title

	if d.Schema.Description != "" {
		if description != "" {
			description += ": "
		}
		description += d.Schema.Description
	}

	constraints := getDescriptionConstraints(d.Schema)
	if constraints != "" {
		if description != "" {
			description += " "
		}
		description += fmt.Sprintf("(%s)", constraints)
	}

	return description
}

func (d SchemaFlagValueDesc) Usage() (usage string) {
	usage = d.Description()

	if shouldRecommendHelpValue(d.Schema) {
		if usage != "" {
			usage += "\n"
		}
		usage += fmt.Sprintf("Use --%s=%s for more details", d.FlagName, ValueHelpIsRequired)
	}

	return
}
