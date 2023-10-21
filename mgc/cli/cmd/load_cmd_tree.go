package cmd

import (
	"context"
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"golang.org/x/exp/slices"
	"magalu.cloud/core"
	mgcSdk "magalu.cloud/sdk"
)

func addChildDesc(sdk *mgcSdk.Sdk, parentCmd *cobra.Command, child core.Descriptor) (*cobra.Command, core.Descriptor, error) {
	if childGroup, ok := child.(mgcSdk.Grouper); ok {
		cmd, err := addGroup(sdk, parentCmd, childGroup)
		return cmd, childGroup, err
	} else if childExec, ok := child.(mgcSdk.Executor); ok {
		cmd, err := addAction(sdk, parentCmd, childExec)
		return cmd, childExec, err
	} else {
		return nil, nil, fmt.Errorf("child %v not group/executor", child)
	}
}

func loadChild(sdk *mgcSdk.Sdk, cmd *cobra.Command, cmdDesc core.Descriptor, childName string) (*cobra.Command, core.Descriptor, error) {
	grouper, ok := cmdDesc.(core.Grouper)
	if !ok {
		return nil, nil, nil
	}

	child, err := grouper.GetChildByName(childName)
	if err != nil {
		return nil, nil, err
	}

	return addChildDesc(sdk, cmd, child)
}

func loadAllChildren(sdk *mgcSdk.Sdk, cmd *cobra.Command, cmdDesc core.Descriptor) (bool, error) {
	grouper, ok := cmdDesc.(core.Grouper)
	if !ok {
		return false, nil
	}

	return grouper.VisitChildren(func(child core.Descriptor) (run bool, err error) {
		if child.IsInternal() && !getShowInternalFlag(cmd.Root()) {
			return true, nil
		}
		_, _, err = addChildDesc(sdk, cmd, child)
		return true, err
	})
}

func loadCommandTree(sdk *mgcSdk.Sdk, cmd *cobra.Command, cmdDesc core.Descriptor, args []string) error {
	childName, childArgs := getNextUnknownCommand(cmd, args)
	if childName == nil || *childName == "help" {
		_, err := loadAllChildren(sdk, cmd, cmdDesc)
		return err
	}

	childCmd, childCmdDesc, err := loadChild(sdk, cmd, cmdDesc, *childName)
	if err != nil {
		// If loading specified child fails, force load all children to print in help command
		// as all available child commands
		if _, loadAllErr := loadAllChildren(sdk, cmd, cmdDesc); loadAllErr != nil {
			return loadAllErr
		}

		return err
	}

	return loadCommandTree(sdk, childCmd, childCmdDesc, childArgs)
}

func addFlags(flags *flag.FlagSet, schema *mgcSdk.Schema) {
	for name, propRef := range schema.Properties {
		prop := propRef.Value
		isRequired := slices.Contains(schema.Required, name)

		propType := getPropType((*mgcSdk.Schema)(prop))
		if propType == "boolean" {
			def, _ := prop.Default.(bool)
			flags.Bool(name, def, prop.Description)
		} else {
			var value any
			if prop.Default != nil {
				value = prop.Default
			}

			constraints := fmt.Sprintf("(%s)", schemaValueConstraints((*mgcSdk.Schema)(prop)))
			description := prop.Description
			if constraints != "()" {
				if description == "" {
					description += constraints
				} else {
					description += fmt.Sprintf(" %s", constraints)
				}
			}

			f := &anyFlagValue{value: value, typeName: propType}
			flags.AddFlag(&flag.Flag{
				Name:     name,
				DefValue: f.String(),
				Usage:    description,
				Value:    f,
			})
		}

		if isRequired {
			if err := cobra.MarkFlagRequired(flags, name); err != nil {
				// Will probably never happen
				logger().Warnw(
					"unable to mark flag as required, but it should be required",
					"flag name", name,
					"error", err.Error(),
				)
			}
		}
	}
}

func addAction(
	sdk *mgcSdk.Sdk,
	parentCmd *cobra.Command,
	exec mgcSdk.Executor,
) (*cobra.Command, error) {
	desc := exec.(mgcSdk.Descriptor)

	actionCmd := &cobra.Command{
		Use:     desc.Name(),
		Short:   desc.Summary(),
		Long:    desc.Description(),
		Version: desc.Version(),

		RunE: func(cmd *cobra.Command, args []string) error {
			parameters := core.Parameters{}
			configs := core.Configs{}

			config := sdk.Config()

			if err := loadDataFromFlags(cmd.Flags(), exec.ParametersSchema(), parameters); err != nil {
				return err
			}

			if err := loadDataFromConfig(config, cmd.PersistentFlags(), exec.ConfigsSchema(), configs); err != nil {
				return err
			}

			ctx := sdk.NewContext()
			result, err := handleExecutor(ctx, cmd, exec, parameters, configs)
			if err != nil {
				return err
			}

			// First chained args structure is MainArgs
			linkChainedArgs := argParser.ChainedArgs()[1:]
			return handleLinkArgs(ctx, cmd, linkChainedArgs, exec.Links(), config, result)
		},
	}

	addFlags(actionCmd.Flags(), exec.ParametersSchema())
	addFlags(actionCmd.PersistentFlags(), exec.ConfigsSchema())

	parentCmd.AddCommand(actionCmd)
	logger().Debugw("Executor added to command tree", "name", exec.Name())
	// TODO: Parse this command's flags right after its creation
	return actionCmd, nil
}

func addGroup(
	sdk *mgcSdk.Sdk,
	parentCmd *cobra.Command,
	group mgcSdk.Grouper,
) (*cobra.Command, error) {
	desc := group.(mgcSdk.Descriptor)
	moduleCmd := &cobra.Command{
		Use:     desc.Name(),
		Short:   desc.Summary(),
		Long:    desc.Description(),
		Version: desc.Version(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	parentCmd.AddCommand(moduleCmd)
	logger().Debugw("Groupper added to command tree", "name", group.Name())
	// TODO: Parse this command's flags right after its creation
	return moduleCmd, nil
}

func addLink(
	ctx context.Context,
	parentCmd *cobra.Command,
	config *mgcSdk.Config,
	originalResult core.Result,
	link core.Linker,
	followingLinkArgs [][]string,
) *cobra.Command {
	linkCmd := &cobra.Command{
		Use:   link.Name(),
		Short: link.Description(),
		RunE: func(cmd *cobra.Command, args []string) error {
			t := table.NewWriter()
			t.AppendHeader(table.Row{"Executing link"})
			t.AppendRows([]table.Row{{"Name", link.Name()}, {"Description", link.Description()}})
			t.SetStyle(table.StyleRounded)
			fmt.Println()
			fmt.Println(t.Render())
			fmt.Println()

			additionalParameters := core.Parameters{}
			additionalConfigs := core.Configs{}

			if err := loadDataFromFlags(cmd.Flags(), link.AdditionalParametersSchema(), additionalParameters); err != nil {
				return err
			}

			if err := loadDataFromConfig(config, cmd.PersistentFlags(), link.AdditionalConfigsSchema(), additionalConfigs); err != nil {
				return err
			}

			exec, err := link.CreateExecutor(originalResult)
			if err != nil {
				return fmt.Errorf("unable to resolve link %s: %w", link.Name(), err)
			}

			result, err := handleExecutor(ctx, cmd, exec, additionalParameters, additionalConfigs)
			if err != nil {
				return err
			}

			return handleLinkArgs(ctx, cmd, followingLinkArgs, exec.Links(), config, result)
		},
	}

	addFlags(linkCmd.Flags(), link.AdditionalParametersSchema())
	addFlags(linkCmd.PersistentFlags(), link.AdditionalConfigsSchema())

	parentCmd.AddCommand(linkCmd)
	logger().Debugw("Link added to command tree", "name", link.Name())

	// Reset values of persistent flags to avoid inheriting the values set from previous actions/links
	linkCmd.PersistentFlags().Visit(func(f *flag.Flag) {
		_ = f.Value.Set(f.DefValue)
	})

	return linkCmd
}
