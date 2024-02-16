package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"magalu.cloud/cli/ui"
	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
	mgcSdk "magalu.cloud/sdk"
)

const (
	setExecNamePrefix    = "set"
	listExecNamePrefix   = "list"
	selectExecNamePrefix = "select"
)

// Let's make the String() more human-friendly, others default to json.
type selectorChoice struct {
	value any
}

func (c selectorChoice) String() string {
	switch v := c.value.(type) {
	case map[string]any:
		s := ""
		for key, value := range v {
			if s != "" {
				s += ", "
			}
			s += fmt.Sprintf("%s=%#v", key, value)
		}
		return s
	default:
		d, err := json.Marshal(v)
		if err == nil {
			return string(d)
		}
		return fmt.Sprint(v)
	}
}

func matchListAndSetExecutor(setExec, listExec core.Executor) bool {
	listSchema := listExec.ResultSchema()
	if listSchema == nil || listSchema.Type != "array" {
		logger().Debugw("List executor does not return an array", "list", listExec, "schema", listSchema)
		return false
	}

	listSchema = (*mgcSchemaPkg.Schema)(listSchema.Items.Value)

	for paramName, paramSchemaRef := range setExec.ParametersSchema().Properties {
		paramSchema := (*mgcSchemaPkg.Schema)(paramSchemaRef.Value)
		if mgcSchemaPkg.CheckSimilarJsonSchemas(paramSchema, listSchema) {
			// list of actual items to be used
			continue
		}

		if listSchema.Type != "object" {
			return false
		}
		fieldSchemaRef := listSchema.Properties[paramName]
		if fieldSchemaRef == nil {
			return false
		}
		if !mgcSchemaPkg.CheckSimilarJsonSchemas(paramSchema, (*mgcSchemaPkg.Schema)(fieldSchemaRef.Value)) {
			return false
		}
	}

	logger().Debugw("List matches the Set executor", "list", listExec, "set", setExec)

	return true
}

func findListForSetExecutor(setExec core.Executor, listExecutors []core.Executor) core.Executor {
	// TODO: maybe use explicit links to annotate that?

	for _, listExec := range listExecutors {
		if matchListAndSetExecutor(setExec, listExec) {
			return listExec
		}
	}

	return nil
}

func loadSelectHelperCommand(sdk *mgcSdk.Sdk, cmd *cobra.Command, cmdGrouper core.Grouper) (err error) {
	var setExecutors []core.Executor
	var listExecutors []core.Executor

	_, err = cmdGrouper.VisitChildren(func(child core.Descriptor) (bool, error) {
		if exec, ok := child.(core.Executor); ok {
			name := exec.Name()
			switch {
			case strings.HasPrefix(name, setExecNamePrefix):
				setExecutors = append(setExecutors, exec)
			case strings.HasPrefix(name, listExecNamePrefix):
				listExecutors = append(listExecutors, exec)
			}
		}
		return true, nil
	})

	if len(setExecutors) == 0 || len(listExecutors) == 0 {
		return
	}

	for _, setExec := range setExecutors {
		if listExec := findListForSetExecutor(setExec, listExecutors); listExec != nil {
			if err = addSelectHelperCommand(sdk, cmd, setExec, listExec); err != nil {
				return
			}
		}
	}

	return
}

func addSelectHelperCommand(sdk *mgcSdk.Sdk, parentCmd *cobra.Command, setExec, listExec core.Executor) (err error) {
	setCmdName, _ := getCommandNameAndAliases(setExec.Name())
	listCmdName, _ := getCommandNameAndAliases(listExec.Name())

	listFlags, err := newExecutorCmdFlags(parentCmd, listExec)
	if err != nil {
		return
	}

	setExecNameSuffix := setExec.Name()[len(setExecNamePrefix):]
	selectName, selectAliases := getCommandNameAndAliases(selectExecNamePrefix + setExecNameSuffix)

	selectCmd := &cobra.Command{
		Use:     selectName,
		Aliases: selectAliases,
		Short:   fmt.Sprintf("call %q, prompt selection and then %q", listCmdName, setCmdName),
		Long:    fmt.Sprintf("helper to interactively call %q, prompt user select one item and call %q with the selection.", listCmdName, setCmdName),

		RunE: func(cmd *cobra.Command, args []string) (err error) {
			config := sdk.Config()
			parameters, configs, err := listFlags.getValues(config, args)
			if err != nil {
				return
			}

			ctx := sdk.NewContext()
			listResult, err := handleExecutorPre(ctx, sdk, cmd, listExec, parameters, configs)
			if err != nil {
				return
			}

			resultWithValue, ok := core.ResultAs[core.ResultWithValue](listResult)
			if !ok {
				return fmt.Errorf("list returned no value")
			}

			resultValue := resultWithValue.Value()
			var resultArray []any
			switch v := resultValue.(type) {
			case []any:
				resultArray = v
			default:
				return fmt.Errorf("list expected to return array, got %T instead: %#v", v, v)
			}

			choices := make([]selectorChoice, len(resultArray))
			for i, v := range resultArray {
				choices[i] = selectorChoice{value: v}
			}

			choice, err := ui.SelectionPrompt(
				fmt.Sprintf("Select entry to be used with %q:", setCmdName),
				choices,
			)
			if err != nil {
				return
			}

			choiceValue := choice.value

			parameters = core.Parameters{}
			listSchema := listExec.ResultSchema().Items.Value // this was checked by matchListAndSetExecutor()
			for paramName, paramSchemaRef := range setExec.ParametersSchema().Properties {
				paramSchema := (*mgcSchemaPkg.Schema)(paramSchemaRef.Value)
				if mgcSchemaPkg.CheckSimilarJsonSchemas(paramSchema, (*mgcSchemaPkg.Schema)(listSchema)) {
					// list of actual items to be used
					parameters[paramName] = choiceValue
					continue
				}
				if m, ok := choiceValue.(map[string]any); ok {
					if value, ok := m[paramName]; ok {
						parameters[paramName] = value
						continue
					}
				}
				logger().Warnw("Missing set parameter from list result", "paramName", paramName, "choice", choice)
			}

			ctx = sdk.NewContext() // use a new context, reset timeouts and the likes
			_, err = handleExecutor(ctx, sdk, cmd, setExec, parameters, configs)
			return err
		},
	}

	if listFlags != nil {
		listFlags.addFlags(selectCmd)
	}

	parentCmd.AddCommand(selectCmd)

	logger().Debugw("Select helper added to command tree", "name", selectName)
	return
}
