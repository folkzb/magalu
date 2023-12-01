package schema_flags

import (
	"encoding/json"
)

type schemaFlagValueString struct {
	schemaFlagValueCommon
}

var _ SchemaFlagValue = (*schemaFlagValueString)(nil)

func newSchemaFlagValueString(desc SchemaFlagValueDesc) *schemaFlagValueString {
	return &schemaFlagValueString{initSchemaFlagValueCommon(desc)}
}

func (o *schemaFlagValueString) unescapedIfNeeded(raw string) string {
	if raw == "" {
		return ""
	}

	// pflag will format "string" as %q, which will escape exiting quotes
	if true { // TODO: next commit raw[0] == '"' && o.Type() == "string" {
		var s string
		if json.Unmarshal([]byte(raw), &s) == nil {
			return s
		}
	}

	// everything else (enum, uri...) can be returned with the quotes
	return raw
}

func (o *schemaFlagValueString) String() string {
	return o.unescapedIfNeeded(o.rawValue)
}

func (o *schemaFlagValueString) RawDefaultValue() string {
	return o.unescapedIfNeeded(o.desc.RawDefaultValue())
}

func (o *schemaFlagValueString) Parse() (value any, err error) {
	return parseStringFlagValue(o.rawValue)
}
