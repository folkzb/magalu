package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"slices"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"magalu.cloud/cli/cmd/schema_flags"
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
	mgcSdk "magalu.cloud/sdk"
)

// It will solve name conflicts between existing flags, parameters and configs,
// will handle "positional arguments" flags and more.
//
// each flag.Value is guaranteed to be schema_flags.SchemaFlagValue

type cmdFlags struct {
	schemaFlags    []*flag.Flag // only schema_flags.SchemaFlagValue elements
	positionalArgs []*flag.Flag // subset schemaFlags that can be positional, in order
	extraFlags     []*flag.Flag

	knownFlags map[flag.NormalizedName]*flag.Flag // all known flags, both existing and schemaFlags
}

// Public Methods:

// these are the public/external/user-visible names
func (cf *cmdFlags) positionalArgsNames() (names []string) {
	if len(cf.positionalArgs) == 0 {
		return
	}

	names = make([]string, len(cf.positionalArgs))
	for i, f := range cf.positionalArgs {
		names[i] = f.Name
	}

	return
}

func (cf *cmdFlags) positionalArgsFunction(cmd *cobra.Command, args []string) (err error) {
	// TODO: if we have a single array in cf.positionalArgs, create positionalArgs [...array], considering it's minimum/maximum items
	// this is useful for "cp src1 ... srcN dst"
	if len(args) > len(cf.positionalArgs) {
		return fmt.Errorf("accepts at most %d arg(s), received %d", len(cf.positionalArgs), len(args))
	}

	for i, value := range args {
		f := cf.positionalArgs[i]
		if err = f.Value.Set(value); err != nil {
			err = fmt.Errorf("invalid argument for %s: %s", f.Name, err.Error())
			return
		}
	}

	return
}

func completeEnum(f *flag.Flag, toComplete string, completions []string) []string {
	fv, ok := f.Value.(schema_flags.SchemaFlagValue)
	if !ok {
		return completions
	}

	var prefixMatches, containsMatches, nonMatches []string

	for _, v := range fv.Desc().Schema.Enum {
		s, ok := v.(string)
		if !ok {
			if data, err := json.Marshal(v); err != nil {
				s = string(data)
			}
		}
		if strings.HasPrefix(s, toComplete) {
			prefixMatches = append(prefixMatches, s)
		} else if strings.Contains(s, toComplete) {
			containsMatches = append(containsMatches, s)
		} else {
			nonMatches = append(nonMatches, s)
		}
	}

	if len(prefixMatches) > 0 {
		return append(completions, prefixMatches...)
	}
	if len(containsMatches) > 0 {
		return append(completions, containsMatches...)
	}

	return append(completions, nonMatches...)
}

func (cf *cmdFlags) validateArgs(cmd *cobra.Command, args []string, toComplete string) (completions []string, directive cobra.ShellCompDirective) {
	logger().Debug("validateArgs", "cmd", cmd.Use, "args", args, "toComplete", toComplete)

	directive = cobra.ShellCompDirectiveNoFileComp
	if len(cf.positionalArgs) == 0 {
		completions = cobra.AppendActiveHelp(completions, "This command does not take any arguments")
		return
	}
	if len(args) >= len(cf.positionalArgs) {
		completions = cobra.AppendActiveHelp(completions, "This command does not take any more arguments")
		return
	}

	f := cf.positionalArgs[len(args)]

	return cf.completeFlag(f, cmd, args, toComplete, completions)
}

func getFlagActiveHelp(f *flag.Flag) string {
	if description := getFlagDescription(f); description != "" {
		return fmt.Sprintf("%s (%s)", f.Name, description)
	}
	return f.Name
}

func (cf *cmdFlags) completeFlag(f *flag.Flag, cmd *cobra.Command, args []string, toComplete string, completions []string) ([]string, cobra.ShellCompDirective) {
	var directive cobra.ShellCompDirective

	completions = cobra.AppendActiveHelp(completions, getFlagActiveHelp(f))

	switch f.Value.Type() {
	case "enum":
		completions = completeEnum(f, toComplete, completions)
		directive = cobra.ShellCompDirectiveNoFileComp

	case schema_flags.FlagTypeFile:
		directive = cobra.ShellCompDirectiveDefault

	case schema_flags.FlagTypeDirectory:
		directive = cobra.ShellCompDirectiveFilterDirs

	default:
		if f.DefValue != "" {
			completions = append(completions, f.DefValue)
		}
		if strings.HasPrefix(toComplete, schema_flags.ValueLoadJSONFromFilePrefix) || strings.HasPrefix(toComplete, schema_flags.ValueLoadVerbatimFromFilePrefix) {
			directive = cobra.ShellCompDirectiveDefault
		} else {
			directive = cobra.ShellCompDirectiveNoFileComp
		}
	}

	return completions, directive
}

func (cf *cmdFlags) newCompleteFlagFunc(f *flag.Flag) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cf.completeFlag(f, cmd, args, toComplete, nil)
	}
}

func (cf *cmdFlags) addFlags(cmd *cobra.Command) {
	configFlags := cmd.Root().PersistentFlags()
	parametersFlags := cmd.Flags()

	for _, f := range cf.schemaFlags {
		var flags *flag.FlagSet
		desc := f.Value.(schema_flags.SchemaFlagValue).Desc()
		if desc.IsConfig {
			flags = configFlags
		} else {
			flags = parametersFlags
		}
		logger().Debugw("adding schema flag", "flag", f.Name, "desc", desc)
		flags.AddFlag(f)
		_ = cmd.RegisterFlagCompletionFunc(f.Name, cf.newCompleteFlagFunc(f))
	}

	for _, f := range cf.extraFlags {
		logger().Debugw("adding extra flag", "flag", f.Name, "value", f.Value)
		parametersFlags.AddFlag(f)
		_ = cmd.RegisterFlagCompletionFunc(f.Name, cf.newCompleteFlagFunc(f))
	}
}

func (cf *cmdFlags) addExtraFlag(f *flag.Flag) {
	cf.knownFlags[flag.NormalizedName(f.Name)] = f
	cf.extraFlags = append(cf.extraFlags, f)
}

// parse, then validate flags and return the final values.
//
// flags that can be positional will be loaded from `argValues`,
// flags that were not set but could be set via configuration, will
// be loaded from `config`.
func (cf *cmdFlags) getValues(config *mgcSdk.Config, argValues []string) (
	parameters core.Parameters,
	configs core.Configs,
	err error,
) {
	parameters = core.Parameters{}
	configs = core.Configs{}

	var loadErrors utils.MultiError
	var missingRequiredFlags requiredFlagsError

	for _, f := range cf.schemaFlags {
		var value any
		value, err = schema_flags.GetFlagValue(f, config)
		logger().Debugw("parsed flag", "flag", f.Name, "desc", f.Value.(schema_flags.SchemaFlagValue).Desc(), "value", value, "error", err)
		if err == schema_flags.ErrNoFlagValue {
			continue
		} else if err == schema_flags.ErrRequiredFlag {
			missingRequiredFlags = append(missingRequiredFlags, f)
		} else if err == schema_flags.ErrWantHelp {
			showFlagHelp(f)
			return
		} else if err != nil {
			loadErrors = append(loadErrors, &flagError{Flag: f, Err: err})
		} else {
			desc := f.Value.(schema_flags.SchemaFlagValue).Desc()
			if desc.IsConfig {
				configs[desc.PropName] = value
			} else {
				parameters[desc.PropName] = value
			}
		}
	}

	if len(missingRequiredFlags) > 0 {
		loadErrors = append(loadErrors, missingRequiredFlags)
	}

	if len(loadErrors) > 0 {
		err = core.UsageError{Err: loadErrors}
	}

	return
}

func newCmdFlags(
	parentCmd *cobra.Command, // used to discover existing flags
	parametersSchema, configsSchema *mgcSdk.Schema,
	positionalArgs []string, // names must match parameterSchema.Properties keys
) (cf *cmdFlags, err error) {
	schemaFlagsLen := len(parametersSchema.Properties) + len(configsSchema.Properties)

	cf = &cmdFlags{
		knownFlags:  make(map[flag.NormalizedName]*flag.Flag, schemaFlagsLen),
		schemaFlags: make([]*flag.Flag, 0, schemaFlagsLen),
	}

	parentFlags := parentCmd.Flags()
	parentFlags.VisitAll(cf.addExistingFlag)
	parentCmd.Root().Flags().VisitAll(cf.addExistingFlag)

	normalizeFunc := parentFlags.GetNormalizeFunc()
	normalizeName := func(name string) flag.NormalizedName {
		return normalizeFunc(parentFlags, name)
	}

	err = cf.addParametersFlags(parametersSchema, positionalArgs, normalizeName)
	cf.addConfigsFlags(configsSchema, normalizeName)

	return
}

func newExecutorCmdFlags(parentCmd *cobra.Command, exec core.Executor) (*cmdFlags, error) {
	return newCmdFlags(
		parentCmd,
		exec.ParametersSchema(),
		exec.ConfigsSchema(),
		exec.PositionalArgs(),
	)
}

// Internal Methods:

func (cf *cmdFlags) addExistingFlag(existingFlag *flag.Flag) {
	cf.knownFlags[flag.NormalizedName(existingFlag.Name)] = existingFlag
}

func (cf *cmdFlags) addSchemaFlag(
	container *mgcSdk.Schema,
	propName string,
	conflictPrefix flag.NormalizedName, // used if propName already exists
	normalizeName func(name string) flag.NormalizedName,
	isRequired bool,
	isConfig bool,
) (f *flag.Flag) {
	flagName := normalizeName(propName)
	for cf.knownFlags[flagName] != nil {
		flagName = conflictPrefix + flagName
	}

	f = schema_flags.NewSchemaFlag(
		container,
		propName,
		flagName,
		isRequired,
		isConfig,
	)
	cf.knownFlags[flagName] = f
	cf.schemaFlags = append(cf.schemaFlags, f)

	return
}

func (cf *cmdFlags) addParametersFlags(
	parametersSchema *mgcSdk.Schema,
	positionalArgs []string,
	normalizeName func(name string) flag.NormalizedName,
) error {
	if len(positionalArgs) > 0 {
		cf.positionalArgs = make([]*flag.Flag, len(positionalArgs))
	}

	for propName := range parametersSchema.Properties {
		f := cf.addSchemaFlag(
			parametersSchema,
			propName,
			normalizeName("param."),
			normalizeName,
			slices.Contains(parametersSchema.Required, propName),
			false,
		)
		position := slices.Index(positionalArgs, propName)
		if position >= 0 {
			cf.positionalArgs[position] = f
		}
	}

	for i, f := range cf.positionalArgs {
		if f == nil {
			// these must not happen in practice, unless we did a mistake in our sdk (static, blueprint, openapi)
			return fmt.Errorf("programming error: positionalArgs[%d] %q is not an existing schema property", i, positionalArgs[i])
		}
	}

	return nil
}

func (cf *cmdFlags) addConfigsFlags(
	configsSchema *mgcSdk.Schema,
	normalizeName func(name string) flag.NormalizedName,
) {
	for propName := range configsSchema.Properties {
		_ = cf.addSchemaFlag(
			configsSchema,
			propName,
			normalizeName("config."),
			normalizeName,
			false,
			true,
		)
	}
}
