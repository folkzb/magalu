package config

import (
	"context"
	"fmt"

	"magalu.cloud/core"
	mgcConfigPkg "magalu.cloud/core/config"
	"magalu.cloud/core/utils"
)

type configGetParams struct {
	Key string `json:"key" validate:"required" jsonschema_description:"Name of the desired config" mgc:"positional"`
}

var getGet = utils.NewLazyLoader[core.Executor](newGet)

func newGet() core.Executor {
	return core.NewStaticExecute(
		core.DescriptorSpec{
			Name:    "get",
			Summary: "Get a specific Config value that has been previously set",
			Description: `Get a specific Config value that has been previously set. If there's an env variable
matching the key (in uppercase and with the 'MGC_' prefix), it'll be retreived.
Otherwise, the value will be searched for in the YAML file`,
		},
		func(ctx context.Context, parameter configGetParams, _ struct{}) (result core.Value, err error) {
			config := mgcConfigPkg.FromContext(ctx)
			if config == nil {
				return nil, fmt.Errorf("unable to retrieve system configuration")
			}
			var out any
			if err := config.Get(parameter.Key, &out); err != nil {
				return nil, err
			}

			return out, nil
		},
	)
}
