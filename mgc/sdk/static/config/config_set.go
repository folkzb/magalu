package config

import (
	"context"
	"fmt"

	"magalu.cloud/core"
)

type configSetParams struct {
	Key   string `jsonschema_description:"Name of the desired config"`
	Value any    `jsonschema_description:"New flag value"`
}

func newConfigSet() *core.StaticExecute {
	return core.NewStaticExecute(
		"set",
		"",
		"Sets a specific config value",
		func(ctx context.Context, parameter configSetParams, _ struct{}) (result core.Value, err error) {
			config := core.ConfigFromContext(ctx)
			if config == nil {
				return nil, fmt.Errorf("unable to retrieve system configuration")
			}

			root := core.GrouperFromContext(ctx)
			if root == nil {
				return nil, fmt.Errorf("unable to retrieve Group from context")
			}

			finished, err := core.VisitAllExecutors(root, []string{}, func(executor core.Executor, path []string) (bool, error) {
				for name, ref := range executor.ConfigsSchema().Properties {
					if name == parameter.Key {
						schema := ref.Value

						if err := schema.VisitJSON(parameter.Value); err != nil {
							return false, err
						}

						if err := config.Set(parameter.Key, parameter.Value); err != nil {
							return false, err
						}

						return false, nil
					}
				}

				return true, nil
			})

			if finished {
				return nil, fmt.Errorf("no config %s found", parameter.Key)
			}

			if err != nil {
				return nil, err
			}

			return parameter.Value, nil
		},
	)
}
