package config

import (
	"github.com/invopop/jsonschema"
	"magalu.cloud/core"
)

var networkConfigSchema *core.Schema

type NetworkConfig struct {
	ServerUrl string `json:"serverUrl,omitempty" jsonschema:"description=Manually specify the server to use,format=uri,default="`
}

func NetworkConfigSchema() *core.Schema {
	if networkConfigSchema == nil {
		networkConfigSchema, _ = core.ToCoreSchema(jsonschema.Reflect(NetworkConfig{}))
	}
	return networkConfigSchema
}
