package cmd

import (
	"context"
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"magalu.cloud/core"
	mgcSdk "magalu.cloud/sdk"
)

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
		linkCmd := addLink(ctx, parentCmd, config, originalResult, link, linkChainedArgs[1:])
		err := linkCmd.ParseFlags(currentLinkArgs[1:])
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
			return fmt.Errorf("Invalid link execution. Command %q doesn't support any links.", parentCmd.Use)
		} else {
			return fmt.Errorf("Invalid link execution. Command %q has no link %q. Available links are %v", parentCmd.Use, linkName, allLinkNames)
		}
	}
}
