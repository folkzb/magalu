package config

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"text/tabwriter"

	"magalu.cloud/core"
)

func NewGroup() *core.StaticGroup {
	return core.NewStaticGroup(
		"config",
		"",
		"config related commands",
		[]core.Descriptor{
			newConfigList(), // cmd: config list
			newConfigGet(),  // cmd: config get
			newConfigSet(),  // cmd: config set
		},
	)
}

func configListFormatter(result core.Value) string {
	configMap := result.(map[string]any) // it must be this, assert

	// TODO: find a better table formatter library
	b := &bytes.Buffer{}
	w := tabwriter.NewWriter(b, 8, 8, 1, ' ', tabwriter.Debug)

	fmt.Fprintf(w, "Config\tType\tDescription\n")
	fmt.Fprintf(w, "---\t---\t---\n")
	for name, schema := range configMap {
		schema := schema.(*core.Schema)
		t := schema.Type // TODO: handle complex types such as enum, array, object...
		fmt.Fprintf(w, "%s\t%s\t%s\n", name, t, schema.Description)
	}
	w.Flush()

	return b.String()
}

func newConfigList() core.Executor {
	executor := core.NewStaticExecuteSimple(
		"list",
		"",
		"list all possible configs",
		func(ctx context.Context) (result map[string]any, err error) {
			root := core.GrouperFromContext(ctx)
			if root == nil {
				return nil, fmt.Errorf("Couldn't get Group from context")
			}

			configMap := map[string]any{}
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
		},
	)

	return core.NewExecuteFormat(executor, configListFormatter)
}
