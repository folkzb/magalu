package config

import (
	"context"
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"magalu.cloud/core"
	mgcConfigPkg "magalu.cloud/core/config"
	mgcSchemaPkg "magalu.cloud/core/schema"
	mgcUtilsPkg "magalu.cloud/core/utils"
)

var listLogger = mgcUtilsPkg.NewLazyLoader(func() *zap.SugaredLogger {
	return logger().Named("list")
})

func configListFormatter(exec core.Executor, result core.Result) string {
	// it must be this, no need to check
	resultWithValue, _ := core.ResultAs[core.ResultWithValue](result)
	configMap := resultWithValue.Value().(map[string]any)

	writer := table.NewWriter()
	writer.AppendHeader(table.Row{"Config", "Type", "Description"})

	sortedKeys := make([]string, 0, len(configMap))
	for k := range configMap {
		sortedKeys = append(sortedKeys, k)
	}
	slices.Sort(sortedKeys)

	for _, k := range sortedKeys {
		schema := configMap[k].(map[string]any)
		writer.AppendRow(table.Row{k, schema["type"], schema["description"]})
	}

	return writer.Render()
}

var getList = mgcUtilsPkg.NewLazyLoader[core.Executor](newList)

func newList() core.Executor {
	executor := core.NewStaticExecuteSimple(
		core.DescriptorSpec{
			Name:        "list",
			Description: "List all available Configs",
		},
		getAllConfigs,
	)
	return core.NewExecuteFormat(executor, configListFormatter)
}

func getAllConfigs(ctx context.Context) (map[string]*core.Schema, error) {
	root := core.GrouperFromContext(ctx)
	if root == nil {
		return nil, fmt.Errorf("couldn't get Group from context")
	}

	config := mgcConfigPkg.FromContext(ctx)
	if config == nil {
		return nil, fmt.Errorf("couldn't get Config from context")
	}

	configMap, err := config.BuiltInConfigs()
	if err != nil {
		return nil, fmt.Errorf("unable to get built in configs: %w", err)
	}

	_, err = core.VisitAllExecutors(root, []string{}, false, func(executor core.Executor, path []string) (bool, error) {
		for name, ref := range executor.ConfigsSchema().Properties {
			current := (*core.Schema)(ref.Value)

			if existing, ok := configMap[name]; ok {
				if err := mgcSchemaPkg.CompareJsonSchemas(existing, current); err != nil {
					listLogger().Warnw("unhandled diverging config", "config", name, "path", path, "current", current, "existing", existing, "error", err)
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
