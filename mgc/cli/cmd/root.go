package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"sdk"

	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"

	flag "github.com/spf13/pflag"
)

// -- BEGIN: create Dynamic Argument Loaders --

// TODO: likely this DynamicArgLoader is not needed anymore,
// I just converted it to use the sdk stuff without checking in detail

type DynamicArgLoader func(cmd *cobra.Command, target *string) (*cobra.Command, DynamicArgLoader, error)

// What we really need so far:
// - no target (default) or a help target: load all
// - else: run specific target (sub command)
func createCommonDynamicArgLoader(
	loadAll func(cmd *cobra.Command) error,
	loadTarget func(target string, cmd *cobra.Command) (*cobra.Command, DynamicArgLoader, error),
) DynamicArgLoader {
	return func(cmd *cobra.Command, target *string) (*cobra.Command, DynamicArgLoader, error) {
		if target == nil {
			return nil, nil, loadAll(cmd)
		}

		return loadTarget(*target, cmd)
	}
}

// -- END: create Dynamic Argument Loaders --

func handleLoaderChild(cmd *cobra.Command, child sdk.Descriptor) (*cobra.Command, DynamicArgLoader, error) {
	if childGroup, ok := child.(sdk.Grouper); ok {
		return AddGroup(cmd, childGroup)
	} else if childExec, ok := child.(sdk.Executor); ok {
		return AddAction(cmd, childExec)
	} else {
		return nil, nil, fmt.Errorf("child %v not group/executor", child)
	}
}

func createGroupLoader(group sdk.Grouper) DynamicArgLoader {
	return createCommonDynamicArgLoader(
		func(cmd *cobra.Command) error {
			_, err := group.VisitChildren(func(child sdk.Descriptor) (bool, error) {
				_, _, err := handleLoaderChild(cmd, child)
				return true, err
			})

			return err
		},
		func(target string, cmd *cobra.Command) (*cobra.Command, DynamicArgLoader, error) {
			child, err := group.GetChildByName(target)

			if err != nil {
				return nil, nil, err
			}

			return handleLoaderChild(cmd, child)
		},
	)
}

func loadParametersIntoCommand(exec sdk.Executor, cmd *cobra.Command) {
	schema := exec.ParametersSchema()
	for name, propRef := range schema.Properties {
		prop := propRef.Value

		value := ""
		if prop.Default != nil {
			str, err := json.Marshal(prop.Default)
			if err == nil {
				value = string(str)
			}
		}

		cmd.Flags().String(name, value, prop.Description)

		if slices.Contains(schema.Required, name) {
			err := cmd.MarkFlagRequired(name)
			if err != nil {
				log.Printf("Error marking %s as required: %s", name, err)
			}
		}
	}
}

func loadParametersFromCommand(cmd *cobra.Command, schema *sdk.Schema, dst map[string]sdk.Value) error {
	if cmd == nil || schema == nil || dst == nil {
		return fmt.Errorf("invalid command or parameter schema")
	}

	for name, ref := range schema.Properties {
		parameter := ref.Value

		flag := cmd.Flags().Lookup(name)
		if flag == nil {
			continue
		}

		str := flag.Value.String()
		var value any

		if parameter.Type == "string" && !strings.HasPrefix(str, "\"") {
			value = str
		} else {
			err := json.Unmarshal([]byte(str), value)
			if err != nil {
				return err
			}
		}

		dst[name] = value
	}

	return nil
}

func AddAction(
	parentCmd *cobra.Command,
	exec sdk.Executor,
) (*cobra.Command, DynamicArgLoader, error) {
	desc := exec.(sdk.Descriptor)

	actionCmd := &cobra.Command{
		Use:   desc.Name(),
		Short: desc.Description(),
		// TODO: Long:    desc.Description,

		RunE: func(cmd *cobra.Command, args []string) error {
			parameters := map[string]sdk.Value{}
			configs := map[string]sdk.Value{}

			schema := exec.ParametersSchema()
			err := loadParametersFromCommand(cmd, schema, parameters)

			if err != nil {
				fmt.Printf("Error when loading flags into command: %v", err)
			}

			// TODO: Load config

			result, err := exec.Execute(parameters, configs)
			fmt.Println("RESULT:", result, err)
			return err
		},
	}

	loadParametersIntoCommand(exec, actionCmd)
	// TODO: load config

	println("\033[1;36mACTION: ADDED CMD:\033[0m", actionCmd.Use)
	parentCmd.AddCommand(actionCmd)
	return actionCmd, nil, nil
}

func runHelpE(cmd *cobra.Command, args []string) error {
	return cmd.Help()
}

func AddGroup(
	parentCmd *cobra.Command,
	group sdk.Grouper,
) (*cobra.Command, DynamicArgLoader, error) {
	desc := group.(sdk.Descriptor)
	moduleCmd := &cobra.Command{
		Use:     desc.Name(),
		Short:   desc.Description(),
		Version: desc.Version(),
		RunE:    runHelpE,
	}

	loader := createGroupLoader(group)

	println("\033[1;34mGROUP: ADDED CMD:\033[0m", moduleCmd.Use)
	parentCmd.AddCommand(moduleCmd)
	return moduleCmd, loader, nil
}

// BEGIN: copied from cobra: command.go

// These are required to get the next unknown command and its arguments without the flags

func hasNoOptDefVal(name string, fs *flag.FlagSet) bool {
	flag := fs.Lookup(name)
	if flag == nil {
		return false
	}
	return flag.NoOptDefVal != ""
}

func shortHasNoOptDefVal(name string, fs *flag.FlagSet) bool {
	if len(name) == 0 {
		return false
	}

	flag := fs.ShorthandLookup(name[:1])
	if flag == nil {
		return false
	}
	return flag.NoOptDefVal != ""
}

func isFlagArg(arg string) bool {
	return ((len(arg) >= 3 && arg[0:2] == "--") ||
		(len(arg) >= 2 && arg[0] == '-' && arg[1] != '-'))
}

// END: copied from cobra: command.go

// this is a modified Traverse() from cobra: command.go
// we need it to get the next command skipping flags
// it's far from ideal, for instance this is a bug:
//
//	$ main-cmd sub-cmd -h sub-sub-cmd
//
// it will take `sub-sub-cmd` as -h argument (but it's not a boolean)
func getNextUnknownCommand(c *cobra.Command, args []string) (*string, []string) {
	inFlag := false

	for i, arg := range args {
		switch {
		// A long flag with a space separated value
		case strings.HasPrefix(arg, "--") && !strings.Contains(arg, "="):
			// TODO: this isn't quite right, we should really check ahead for 'true' or 'false'
			inFlag = !hasNoOptDefVal(arg[2:], c.Flags())
			continue
		// A short flag with a space separated value
		case strings.HasPrefix(arg, "-") && !strings.Contains(arg, "=") && len(arg) == 2 && !shortHasNoOptDefVal(arg[1:], c.Flags()):
			inFlag = true
			continue
		// The value for a flag
		case inFlag:
			inFlag = false
			continue
		// A flag without a value, or with an `=` separated value
		case isFlagArg(arg):
			continue
		}

		return &arg, args[i+1:]
	}

	return nil, args
}

func DynamicLoadCommand(cmd *cobra.Command, args []string, loader DynamicArgLoader) error {
	childCmd, childArgs, err := cmd.Traverse(args)
	if err != nil {
		return err
	}

	if cmd != childCmd {
		return nil
	}

	var childCmdName *string
	for {
		// NOTE: this replicates cmd.Traverse(), but returns the command and the args without flags
		childCmdName, childArgs = getNextUnknownCommand(cmd, childArgs)
		if childCmdName == nil || *childCmdName != "help" {
			break
		}
	}

	childCmd, childLoader, err := loader(cmd, childCmdName)
	if err != nil || childLoader == nil || childCmd == nil {
		return err
	}

	return DynamicLoadCommand(childCmd, childArgs, childLoader)
}

// NOTE: these will be build in their own files, one file per static command
var staticCmds = sdk.NewStaticGroup(
	"Static Commands",
	"12.34",
	"Test static commands",
	[]sdk.Descriptor{
		sdk.NewStaticExecute(
			"static",
			"34.56",
			"static first level",
			// NOTE: these can (should?) be defined in JSON and unmarshal from string
			sdk.NewObjectSchema(
				map[string]*sdk.Schema{
					"param1": sdk.SetDescription(
						sdk.NewNumberSchema(),
						"Example static parameter of type number",
					),
				},
				[]string{},
			),
			&sdk.Schema{},
			func(parameters, configs map[string]sdk.Value) (result sdk.Value, err error) {
				println("TODO: static first level called")
				return nil, nil
			},
		),
		sdk.NewStaticGroup(
			"vpc",
			"",
			"",
			[]sdk.Descriptor{
				sdk.NewStaticGroup(
					"port",
					"",
					"",
					[]sdk.Descriptor{
						sdk.NewStaticExecute(
							"static",
							"",
							"static third level",
							&sdk.Schema{},
							&sdk.Schema{},
							func(parameters, configs map[string]sdk.Value) (result sdk.Value, err error) {
								println("TODO: vpc port static (third level) called")
								return nil, nil
							},
						),
					},
				),
				sdk.NewStaticExecute(
					"static",
					"",
					"static second level",
					&sdk.Schema{},
					&sdk.Schema{},
					func(parameters, configs map[string]sdk.Value) (result sdk.Value, err error) {
						println("TODO: vpc static (second level) called")
						return nil, nil
					},
				),
			},
		),
	},
)

func Execute() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// TODO: this should be in viper + viper.SetDefault()
	openApiDir := filepath.Join(cwd, "openapis")

	rootCmd := &cobra.Command{
		Use:     "cloud",
		Version: "TODO",
		Short:   "CLI tool for OpenAPI integration",
		Long: `This CLI is a dynamic processor of OpenAPI files that
can generate a command line on-demand for Rest manipulation`,
		RunE: runHelpE,
	}
	rootCmd.AddGroup(&cobra.Group{
		ID:    "other",
		Title: "Other commands:",
	})
	rootCmd.SetHelpCommandGroupID("other")
	rootCmd.SetCompletionCommandGroupID("other")

	extensionPrefix := "x-cli"
	openApi := &sdk.OpenApiSource{
		Dir:             openApiDir,
		ExtensionPrefix: &extensionPrefix,
	}

	merge := sdk.NewMergeGroup(
		"MagaLu CLI",
		"1.0",
		"All MagaLu Commands",
		[]sdk.Grouper{
			openApi,
			staticCmds,
		},
	)

	err = DynamicLoadCommand(rootCmd, os.Args[1:], createGroupLoader(merge))
	if err != nil {
		rootCmd.PrintErrln("Warning: loading dynamic arguments:", err)
	}

	return rootCmd.Execute()
}
