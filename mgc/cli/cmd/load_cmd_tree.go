package cmd

import (
	"context"
	"fmt"

	"slices"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/stoewer/go-strcase"
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
	mgcSdk "magalu.cloud/sdk"
)

const (
	listLinksCmd = "cli.list-links"
)

var allExecutorChildren = []string{listLinksCmd}

func addChildDesc(sdk *mgcSdk.Sdk, parentCmd *cobra.Command, child core.Descriptor) (cmd *cobra.Command, flags *cmdFlags, err error) {
	if childGroup, ok := child.(mgcSdk.Grouper); ok {
		cmd, err = addGroup(sdk, parentCmd, childGroup)
		return
	} else if childExec, ok := child.(mgcSdk.Executor); ok {
		return addAction(sdk, parentCmd, childExec)
	} else {
		err = fmt.Errorf("child %v not group/executor", child)
		return
	}
}

func loadGrouperChild(sdk *mgcSdk.Sdk, cmd *cobra.Command, cmdGrouper core.Grouper, childName string) (*cobra.Command, core.Descriptor, error) {
	child, err := cmdGrouper.GetChildByName(childName)
	if err != nil {
		return nil, nil, err
	}

	childCmd, flags, err := addChildDesc(sdk, cmd, child)
	if err != nil {
		return nil, nil, err
	}

	if flags != nil {
		flags.addFlags(childCmd)
	}

	return childCmd, child, nil
}

func loadChild(sdk *mgcSdk.Sdk, cmd *cobra.Command, cmdDesc core.Descriptor, childName string) (*cobra.Command, core.Descriptor, error) {
	if cmdGrouper, ok := cmdDesc.(core.Grouper); ok {
		return loadGrouperChild(sdk, cmd, cmdGrouper, childName)
	} else if cmdExec, ok := cmdDesc.(core.Executor); ok {
		// Manually fail if specified child is invalid
		if !slices.Contains(allExecutorChildren, childName) {
			return nil, nil, fmt.Errorf("command %q has no child named %q", cmd.Name(), childName)
		}

		// Always load all executor children
		return nil, nil, loadAllExecChildren(sdk, cmd, cmdExec)
	}

	return nil, nil, fmt.Errorf("command %q has no child named %q", cmd.Name(), childName)
}

func loadAllGrouperChildren(sdk *mgcSdk.Sdk, cmd *cobra.Command, cmdGrouper core.Grouper) error {
	_, err := cmdGrouper.VisitChildren(func(child core.Descriptor) (run bool, err error) {
		if child.IsInternal() && !getShowInternalFlag(cmd.Root()) {
			return true, nil
		}
		_, _, err = addChildDesc(sdk, cmd, child)
		return true, err
	})
	return err
}

func loadAllExecChildren(sdk *mgcSdk.Sdk, cmd *cobra.Command, cmdExec core.Executor) error {
	addListLinks(cmd, cmdExec)
	return nil
}

func loadAllChildren(sdk *mgcSdk.Sdk, cmd *cobra.Command, cmdDesc core.Descriptor) error {
	if cmdGrouper, ok := cmdDesc.(core.Grouper); ok {
		return loadAllGrouperChildren(sdk, cmd, cmdGrouper)
	} else if cmdExec, ok := cmdDesc.(core.Executor); ok {
		return loadAllExecChildren(sdk, cmd, cmdExec)
	}

	return nil
}

func isExistingCommand(cmd *cobra.Command, name string) bool {
	for _, c := range cmd.Commands() {
		if c.Name() == name || c.HasAlias(name) {
			return true
		}
	}
	return false
}

func loadCommandTree(sdk *mgcSdk.Sdk, cmd *cobra.Command, cmdDesc core.Descriptor, args []string) error {
	if cmd == nil {
		return nil
	}

	var childName *string
	var childArgs = args
	for {
		childName, childArgs = getNextUnknownCommand(cmd, childArgs)
		if childName == nil || *childName != "help" {
			break
		}
	}

	if childName == nil {
		return loadAllChildren(sdk, cmd, cmdDesc)
	} else if isExistingCommand(cmd, *childName) {
		return nil
	}

	childCmd, childCmdDesc, err := loadChild(sdk, cmd, cmdDesc, *childName)
	if err != nil {
		// If loading specified child fails, force load all children to print in help command
		// as all available child commands
		if loadAllErr := loadAllChildren(sdk, cmd, cmdDesc); loadAllErr != nil {
			return loadAllErr
		}

		if _, ok := cmdDesc.(core.Executor); !ok {
			return err
		}
	}

	return loadCommandTree(sdk, childCmd, childCmdDesc, childArgs)
}

func buildUse(name string, args []string) string {
	use := name
	for _, name := range args {
		use += " " + fmt.Sprintf("[%s]", name)
	}
	return use
}

func getCommandNameAndAliases(origName string) (name string, aliases []string) {
	name = strcase.KebabCase(origName)
	if name != origName {
		aliases = append(aliases, origName)
	}
	return
}

func addAction(
	sdk *mgcSdk.Sdk,
	parentCmd *cobra.Command,
	exec mgcSdk.Executor,
) (actionCmd *cobra.Command, flags *cmdFlags, err error) {
	flags, err = newExecutorCmdFlags(parentCmd, exec)
	if err != nil {
		return
	}

	links := exec.Links()

	name, aliases := getCommandNameAndAliases(exec.Name())

	actionCmd = &cobra.Command{
		Use:     buildUse(name, flags.positionalArgsNames()),
		Aliases: aliases,
		Args:    cobra.MaximumNArgs(len(flags.positionalArgs)),
		Short:   exec.Summary(),
		Long:    exec.Description(),
		Version: exec.Version(),
		GroupID: "catalog",

		RunE: func(cmd *cobra.Command, args []string) error {
			config := sdk.Config()
			parameters, configs, err := flags.getValues(config, args)
			if err != nil {
				return err
			}

			ctx := sdk.NewContext()
			result, err := handleExecutor(ctx, sdk, cmd, exec, parameters, configs)
			if err != nil {
				return err
			}

			// First chained args structure is MainArgs
			linkChainedArgs := argParser.ChainedArgs()[1:]
			return handleLinkArgs(ctx, sdk, cmd, linkChainedArgs, links, config, result)
		},
	}

	parentCmd.AddCommand(actionCmd)

	logger().Debugw("Executor added to command tree", "name", exec.Name())

	return
}

func addGroup(
	sdk *mgcSdk.Sdk,
	parentCmd *cobra.Command,
	group mgcSdk.Grouper,
) (*cobra.Command, error) {
	name, aliases := getCommandNameAndAliases(group.Name())
	moduleCmd := &cobra.Command{
		Use:     name,
		Aliases: aliases,
		Short:   group.Summary(),
		Long:    group.Description(),
		Version: group.Version(),
		GroupID: "catalog",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	moduleCmd.AddGroup(&cobra.Group{
		ID:    "catalog",
		Title: "Commands:",
	})

	parentCmd.AddCommand(moduleCmd)
	logger().Debugw("Groupper added to command tree", "name", group.Name())
	// TODO: Parse this command's flags right after its creation
	return moduleCmd, nil
}

func addLink(
	ctx context.Context,
	sdk *mgcSdk.Sdk,
	parentCmd *cobra.Command,
	config *mgcSdk.Config,
	originalResult core.Result,
	link core.Linker,
	followingLinkArgs [][]string,
) (linkCmd *cobra.Command, err error) {
	flags, err := newCmdFlags(
		parentCmd,
		link.AdditionalParametersSchema(),
		link.AdditionalConfigsSchema(),
		nil,
	)
	if err != nil {
		return
	}

	name, aliases := getCommandNameAndAliases(link.Name())

	linkCmd = &cobra.Command{
		Use:     name,
		Aliases: aliases,
		Short:   link.Description(),
		RunE: func(cmd *cobra.Command, args []string) error {
			printLinkExecutionTable(link.Name(), link.Description())

			additionalParameters, additionalConfigs, err := flags.getValues(config, args)
			if err != nil {
				return err
			}

			exec, err := link.CreateExecutor(originalResult)
			if err != nil {
				return fmt.Errorf("unable to resolve link %s: %w", link.Name(), err)
			}

			result, err := handleExecutor(ctx, sdk, cmd, exec, additionalParameters, additionalConfigs)
			if err != nil {
				return err
			}

			return handleLinkArgs(ctx, sdk, cmd, followingLinkArgs, exec.Links(), config, result)
		},
	}

	parentCmd.AddCommand(linkCmd)
	flags.addFlags(linkCmd)

	logger().Debugw("Link added to command tree", "name", link.Name())

	// Reset values of persistent flags to avoid inheriting the values set from previous actions/links
	linkCmd.PersistentFlags().Visit(func(f *flag.Flag) {
		_ = f.Value.Set(f.DefValue)
	})

	return
}

func addListLinks(
	parentCmd *cobra.Command,
	sourceExec core.Executor,
) *cobra.Command {
	type LinkerListEntry struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	type LinkerList []LinkerListEntry

	listLinksCmd := &cobra.Command{
		Use:   listLinksCmd,
		Short: "List all available links for this command",
		RunE: func(cmd *cobra.Command, args []string) error {
			links := sourceExec.Links()
			result := make(LinkerList, 0, len(links))

			for linkName, link := range links {
				result = append(result, LinkerListEntry{Name: linkName, Description: link.Description()})
			}

			simplified, err := utils.SimplifyAny(result)
			if err != nil {
				return err
			}

			return handleSimpleResultValue(simplified, getOutputFlag(cmd))
		},
	}

	parentCmd.AddCommand(listLinksCmd)
	return listLinksCmd
}
