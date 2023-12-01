package schema_flags

import (
	"encoding/json"
	"fmt"
)

// Allows multiple calls to set different values, instead of overriding the previously set
type schemaFlagValueMulti struct {
	schemaFlagValueCommon
	setValues []string
}

func initSchemaFlagValueMulti(desc SchemaFlagValueDesc) (o schemaFlagValueMulti) {
	return schemaFlagValueMulti{initSchemaFlagValueCommon(desc), nil}
}

func (o *schemaFlagValueMulti) String() string {
	if len(o.setValues) == 0 {
		return o.rawValue
	}

	items, err := o.Parse()
	if err != nil {
		return fmt.Sprintf("<Error: %q, setValues: %#v>", err.Error(), o.setValues)
	}

	data, err := json.Marshal(items)
	if err != nil {
		return fmt.Sprintf("<Error: %q, items: %#v>", err.Error(), items)
	}

	return string(data)
}

func (o *schemaFlagValueMulti) Set(rawValue string) error {
	if rawValue != "" {
		o.setValues = append(o.setValues, rawValue)
		o.changed = true
	}
	return nil
}
