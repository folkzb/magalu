package provider

import (
	"magalu.cloud/core"
)

func getConfigs(schema *core.Schema) core.Configs {
	result := core.Configs{}
	for propName, propRef := range schema.Properties {
		prop := (*core.Schema)(propRef.Value)
		if prop.Default != nil {
			result[propName] = prop.Default
		}
	}
	return result
}
