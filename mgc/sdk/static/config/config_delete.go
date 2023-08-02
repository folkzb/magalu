package config

import (
	"context"
	"fmt"

	"magalu.cloud/core"
)

type configDeleteParams struct {
	Key string `jsonschema_description:"Name of the config to be deleted"`
}

func newConfigDelete() *core.StaticExecute {
	return core.NewStaticExecute(
		"delete",
		"",
		"Deletes a key from config file",
		func(ctx context.Context, parameter configDeleteParams, _ struct{}) (result core.Value, err error) {
			config := core.ConfigFromContext(ctx)
			if config == nil {
				return nil, fmt.Errorf("unable to retrieve system configuration")
			}

			if ok := config.IsSet(parameter.Key); !ok {
				return nil, fmt.Errorf("no config %s found", parameter.Key)
			}

			if err := config.Delete(parameter.Key); err != nil {
				return nil, err
			}

			return nil, nil
		},
	)
}
