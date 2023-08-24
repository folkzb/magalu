package config

import (
	"context"
	"fmt"
	"reflect"

	"github.com/jedib0t/go-pretty/v6/table"
	"magalu.cloud/core"
	"magalu.cloud/core/config"
)

func configListFormatter(result core.Value) string {
	configMap := result.(map[string]any) // it must be this, assert

	writer := table.NewWriter()
	writer.AppendHeader(table.Row{"Config", "Type", "Description"})

	for name, schema := range configMap {
		schema := schema.(map[string]any)
		writer.AppendRow(table.Row{name, schema["type"], schema["description"]})
	}

	return writer.Render()
}

func newList() core.Executor {
	executor := core.NewStaticExecuteSimple(
		"list",
		"",
		"list all possible configs",
		getAllConfigs,
	)
	return core.NewExecuteFormat(executor, configListFormatter)
}

func getAllConfigs(ctx context.Context) (map[string]any, error) {
	root := core.GrouperFromContext(ctx)
	if root == nil {
		return nil, fmt.Errorf("Couldn't get Group from context")
	}

	config := config.FromContext(ctx)
	if root == nil {
		return nil, fmt.Errorf("Couldn't get Config from context")
	}

	configMap, err := config.BuiltInConfigs()
	if err != nil {
		return nil, fmt.Errorf("unable to get built in configs: %w", err)
	}

	_, err = core.VisitAllExecutors(root, []string{}, func(executor core.Executor, path []string) (bool, error) {
		for name, ref := range executor.ConfigsSchema().Properties {
			current := (*core.Schema)(ref.Value)

			if existing, ok := configMap[name]; ok {
				if !reflect.DeepEqual(existing, current) {
					fmt.Printf("WARNING: unhandled diverging config at %v %v: %v != %v\n", path, name, existing, current)
				}

				continue
			}
			configMap[name] = current
		}

		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return configMap, nil
}
