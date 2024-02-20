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

func findListSchema(schema *mgcSchemaPkg.Schema) (resourceSchema *mgcSchemaPkg.Schema, err error) {
	if schema == nil {
		err = fmt.Errorf("missing schema")
		return
	}

	if schema.Items != nil {
		resourceSchema = (*mgcSchemaPkg.Schema)(schema.Items.Value)
		return
	}
	// If the schema is an object and have only one field that is an array
	if len(schema.Properties) == 1 {
		for _, propRef := range schema.Properties {
			prop := propRef.Value

			if prop.Items != nil {
				resourceSchema = (*mgcSchemaPkg.Schema)(prop.Items.Value)
				return
			}
		}
	}

	// If no array schema is found in properties
	err = fmt.Errorf("unable to find resource schema from list result schema")
	return
}

func matchListAndSetExecutor(setExec, listExec core.Executor) (matchingListExec core.Executor, multiple bool) {
	listSchema, err := findListSchema(listExec.ResultSchema())
	if err != nil {
		logger().Debugw("List executor does not return an array", "listSchema", listExec.ResultSchema(), "error", err)
		return
	}

	for paramName, paramSchemaRef := range setExec.ParametersSchema().Properties {
		paramSchema := (*mgcSchemaPkg.Schema)(paramSchemaRef.Value)
		if paramSchema.Type == "array" && listSchema.Type != "array" {
			// allow multiple selection of items
			multiple = true
			paramSchema = (*mgcSchemaPkg.Schema)(paramSchema.Items.Value)
		}

		if mgcSchemaPkg.CheckSimilarJsonSchemas(paramSchema, listSchema) {
			// list of actual items to be used
			continue
		}

		if listSchema.Type != "object" {
			return
		}
		fieldSchemaRef := listSchema.Properties[paramName]
		if fieldSchemaRef == nil {
			return
		}
		if !mgcSchemaPkg.CheckSimilarJsonSchemas(paramSchema, (*mgcSchemaPkg.Schema)(fieldSchemaRef.Value)) {
			return
		}
	}

	logger().Debugw("List matches the Set executor", "list", listExec, "set", setExec)
	matchingListExec = listExec
	return
}

func findListForSetExecutor(setExec core.Executor, listExecutors []core.Executor) (listExec core.Executor, multiple bool) {
	// TODO: maybe use explicit links to annotate that?

	for _, exec := range listExecutors {
		listExec, multiple = matchListAndSetExecutor(setExec, exec)
		if listExec != nil {
			return
		}
	}

	return
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
		if listExec, multiple := findListForSetExecutor(setExec, listExecutors); listExec != nil {
			if err = addSelectHelperCommand(sdk, cmd, setExec, listExec, multiple); err != nil {
				return
			}
		}
	}

	return
}

func getChoiceValue(choice selectorChoice, paramName string, paramSchema, listSchema *mgcSchemaPkg.Schema) (any, bool) {
	if mgcSchemaPkg.CheckSimilarJsonSchemas(paramSchema, (*mgcSchemaPkg.Schema)(listSchema)) {
		// list of actual items to be used
		return choice.value, true
	}

	if m, ok := choice.value.(map[string]any); ok {
		if value, ok := m[paramName]; ok {
			return value, ok
		}
	}

	return nil, false
}

func getMultiChoiceValue(choices []selectorChoice, paramName string, paramSchema, listSchema *mgcSchemaPkg.Schema) (any, bool) {
	if paramSchema.Type == "array" && listSchema.Type != "array" {
		paramSchema = (*mgcSchemaPkg.Schema)(paramSchema.Items.Value)
		lst := make([]any, 0, len(choices))
		for _, c := range choices {
			if value, ok := getChoiceValue(c, paramName, paramSchema, listSchema); ok {
				lst = append(lst, value)
			}
		}
		return lst, true
	}

	for _, c := range choices {
		if value, ok := getChoiceValue(c, paramName, paramSchema, listSchema); ok {
			return value, true
		}
	}

	return nil, false
}

func selectMultipleAndSetupParameters(
	setCmdName string,
	setExec, listExec core.Executor,
	choices []selectorChoice,
) (parameters core.Parameters, err error) {
	selection, err := ui.MultiSelectionPrompt(
		fmt.Sprintf("Select multiple entries to be used with %q:", setCmdName),
		choices,
	)
	if err != nil {
		return
	}

	parameters = core.Parameters{}
	listSchema := (*mgcSchemaPkg.Schema)(listExec.ResultSchema().Items.Value) // this was checked by matchListAndSetExecutor()
	for paramName, paramSchemaRef := range setExec.ParametersSchema().Properties {
		paramSchema := (*mgcSchemaPkg.Schema)(paramSchemaRef.Value)
		if value, ok := getMultiChoiceValue(selection, paramName, paramSchema, listSchema); ok {
			parameters[paramName] = value
		} else {
			logger().Warnw(
				"Missing set parameter from list result (multiple choices)",
				"paramName", paramName,
				"selection", selection,
				"paramSchema", paramSchema,
				"listSchema", listSchema,
			)
		}
	}

	return
}

func selectOneAndSetupParameters(
	setCmdName string,
	setExec, listExec core.Executor,
	choices []selectorChoice,
) (parameters core.Parameters, err error) {
	choice, err := ui.SelectionPrompt(
		fmt.Sprintf("Select one entry to be used with %q:", setCmdName),
		choices,
	)
	if err != nil {
		return
	}

	parameters = core.Parameters{}
	listSchema := (*mgcSchemaPkg.Schema)(listExec.ResultSchema().Items.Value) // this was checked by matchListAndSetExecutor()
	for paramName, paramSchemaRef := range setExec.ParametersSchema().Properties {
		paramSchema := (*mgcSchemaPkg.Schema)(paramSchemaRef.Value)
		if value, ok := getChoiceValue(choice, paramName, paramSchema, listSchema); ok {
			parameters[paramName] = value
		} else {
			logger().Warnw(
				"Missing set parameter from list result",
				"paramName", paramName,
				"choice", choice.value,
				"paramSchema", paramSchema,
				"listSchema", listSchema,
			)
		}
	}

	return
}

func addSelectHelperCommand(sdk *mgcSdk.Sdk, parentCmd *cobra.Command, setExec, listExec core.Executor, multiple bool) (err error) {
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

			if multiple {
				parameters, err = selectMultipleAndSetupParameters(setCmdName, setExec, listExec, choices)
			} else {
				parameters, err = selectOneAndSetupParameters(setCmdName, setExec, listExec, choices)
			}
			if err != nil {
				return
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
