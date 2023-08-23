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

func newSet() *core.StaticExecute {
	return core.NewStaticExecute(
		"set",
		"",
		"Sets a specific config value",
		func(ctx context.Context, parameter configSetParams, _ struct{}) (core.Value, error) {
			config := core.ConfigFromContext(ctx)
			if config == nil {
				return nil, fmt.Errorf("unable to retrieve system configuration")
			}

			allConfigs, err := getAllConfigs(ctx)
			if err != nil {
				return nil, fmt.Errorf("error when getting possible configs: %w", err)
			}

			schema, ok := allConfigs[parameter.Key]
			if !ok {
				return nil, fmt.Errorf("no config %s found", parameter.Key)
			}

			s, ok := schema.(*core.Schema)
			if !ok {
				// Should never happen
				return nil, fmt.Errorf("no config %s found", parameter.Key)
			}

			if err := s.VisitJSON(parameter.Value); err != nil {
				return nil, err
			}

			if err := config.Set(parameter.Key, parameter.Value); err != nil {
				return nil, err
			}

			return parameter.Value, nil

		},
	)
}
