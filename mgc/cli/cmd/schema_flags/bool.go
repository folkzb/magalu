package schema_flags

type schemaFlagValueBool struct {
	schemaFlagValueCommon
}

var _ SchemaFlagValue = (*schemaFlagValueBool)(nil)

func newSchemaFlagValueBool(desc SchemaFlagValueDesc) *schemaFlagValueBool {
	return &schemaFlagValueBool{initSchemaFlagValueCommon(desc)}
}

func (o *schemaFlagValueBool) Type() string {
	return "bool" // pflag has special handling for it
}

func (o *schemaFlagValueBool) RawDefaultValue() string {
	defVal := o.schemaFlagValueCommon.RawDefaultValue()
	if defVal == "" {
		defVal = "false" // pflag does special check to get the default boolean value
	}
	return defVal
}

func (o *schemaFlagValueBool) RawNoOptDefVal() string {
	return "true" // mimics pflag's boolValue
}

func (o *schemaFlagValueBool) Parse() (value any, err error) {
	return parseBoolFlagValue(o.rawValue)
}

// mimics pflag's boolValue
func (b *schemaFlagValueBool) IsBoolFlag() bool {
	return true
}
