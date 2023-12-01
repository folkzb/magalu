package schema_flags

type schemaFlagValueObject struct {
	schemaFlagValueMulti
}

var _ SchemaFlagValue = (*schemaFlagValueObject)(nil)

func newSchemaFlagValueObject(desc SchemaFlagValueDesc) *schemaFlagValueObject {
	return &schemaFlagValueObject{
		initSchemaFlagValueMulti(desc),
	}
}

func (o *schemaFlagValueObject) Parse() (value any, err error) {
	if len(o.setValues) == 0 {
		return parseObjectFlagValueSingle(o.desc.Schema, o.rawValue)
	}

	return parseObjectFlagValue(o.desc.Schema, o.setValues)
}
