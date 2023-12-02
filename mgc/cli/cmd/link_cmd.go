package cmd

import (
	"context"
	"fmt"

	flag "github.com/spf13/pflag"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"magalu.cloud/cli/cmd/schema_flags"
	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
	mgcSdk "magalu.cloud/sdk"
)

func newListLinkFlag() (f *flag.Flag) {
	const listLinkFlag = "cli.list-links"
	listLinkFlagSchema := mgcSchemaPkg.NewStringSchema()
	listLinkFlagSchema.Description = "List all available links for this command"
	listLinkFlagSchema.Enum = []any{
		"table",
		"json",
		"yaml",
	}

	f = schema_flags.NewSchemaFlag(
		mgcSchemaPkg.NewObjectSchema(map[string]*mgcSchemaPkg.Schema{
			listLinkFlag: listLinkFlagSchema,
		}, nil),
		listLinkFlag,
		listLinkFlag,
		false,
		false,
	)
	f.NoOptDefVal = "table"

	return
}

func listLinks(f *flag.Flag, links core.Links) (err error, used bool) {
	if f == nil {
		return
	}

	output := f.Value.String()
	if output == "" {
		return
	}

	used = true

	type LinkerListEntry struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	type LinkerList []LinkerListEntry

	result := make(LinkerList, 0, len(links))

	for linkName, link := range links {
		result = append(result, LinkerListEntry{Name: linkName, Description: link.Description()})
	}

	simplified, err := utils.SimplifyAny(result)
	if err != nil {
		return
	}

	err = handleSimpleResultValue(simplified, output)
	return
}

func printLinkExecutionTable(name, description string) {
	t := table.NewWriter()
	t.AppendHeader(table.Row{"Executing link"})
	t.AppendRows([]table.Row{{"Name", name}, {"Description", description}})
	t.SetStyle(table.StyleRounded)
	fmt.Println()
	fmt.Println(t.Render())
	fmt.Println()
}

func addLinkHelp(
	parentCmd *cobra.Command,
) *cobra.Command {
	linkHelpCmd := &cobra.Command{
		Use:   "help",
		Short: "Get help on the usage of link chains",
		Run: func(cmd *cobra.Command, args []string) {
			printLinkExecutionTable(cmd.Use, cmd.Short)

			msg := `All Executors might have possible links to other Executors.
These Links use the output of said Executor to aid in deciding which
parameters and configs to pass to the next Executor without needing
to declare them explicitly.

Calling these links is as simple as adding a "!" to the end of a
command call, specifying the link name and passing in any additional
parameters via flags. For instance:

./cli initial command --some-flag flag-value ! link --other-flag other

In this case, 'link' may have access to '--some-flag' (and other internal
metadata), which can aid in its execution following the 'initial command'
execution.`
			fmt.Println(msg)
		},
	}
	parentCmd.AddCommand(linkHelpCmd)
	logger().Debugw("Link added to command tree", "name", "help")
	return linkHelpCmd
}

func handleLinkArgs(
	ctx context.Context,
	sdk *mgcSdk.Sdk,
	parentCmd *cobra.Command,
	linkChainedArgs [][]string,
	links core.Links,
	config *mgcSdk.Config,
	originalResult core.Result,
) error {
	if len(linkChainedArgs) == 0 {
		return nil
	}

	currentLinkArgs := linkChainedArgs[0]
	linkName := currentLinkArgs[0]

	if link, ok := links[linkName]; ok {
		linkCmd, err := addLink(ctx, sdk, parentCmd, config, originalResult, link, linkChainedArgs[1:])
		if err != nil {
			return err
		}
		err = linkCmd.ParseFlags(currentLinkArgs[1:])
		if err != nil {
			return err
		}
		return linkCmd.RunE(linkCmd, []string{})
	} else if linkName == "help" {
		linkHelpCmd := addLinkHelp(parentCmd)
		linkHelpCmd.Run(linkHelpCmd, nil)
		return nil
	} else {
		allLinkNames := make([]string, 0, len(links))
		for linkName := range links {
			allLinkNames = append(allLinkNames, linkName)
		}
		if len(allLinkNames) == 0 {
			return fmt.Errorf("invalid link execution. Command %q doesn't support any links", parentCmd.Use)
		} else {
			return fmt.Errorf("invalid link execution. Command %q has no link %q. Available links are %v", parentCmd.Use, linkName, allLinkNames)
		}
	}
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
