package cmd

import (
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

type cmdLinks struct {
	sdk     *mgcSdk.Sdk
	exec    core.Executor
	cmdPath string

	flag *flag.Flag

	cmdFlags map[string]*cmdFlags
	root     *cobra.Command
}

func newCmdLinks(sdk *mgcSdk.Sdk, exec core.Executor, cmdPath string) (c *cmdLinks) {
	if len(exec.Links()) == 0 {
		logger().Debugw("executor has no links", "exec", exec, "cmdPath", cmdPath)
		return nil
	}

	c = &cmdLinks{
		sdk:      sdk,
		exec:     exec,
		cmdPath:  cmdPath,
		flag:     newListLinkFlag(),
		cmdFlags: map[string]*cmdFlags{},
	}
	c.initCommands()
	logger().Debugw("executor with links", "exec", exec, "cmdPath", cmdPath)

	return
}

func (c *cmdLinks) check(chainedArgs [][]string) (err error, stop bool) {
	if c == nil {
		return
	}

	if err, stop = listLinks(c.flag, c.exec.Links()); err != nil || stop {
		return
	}

	if len(chainedArgs) == 0 {
		return
	}

	args := chainedArgs[0]
	if len(args) == 0 {
		return
	}

	c.root.SetArgs(args)
	err = c.root.Execute() // safe: only help is an actual command, whatever else is just an empty placeholder
	if err == schema_flags.ErrWantHelp {
		err = nil
		stop = true
	}
	logger().Debugw("checked executor link", "args", args, "error", err, "exec", c.exec, "links", c.exec.Links())

	return
}

func (c *cmdLinks) findCommandLink(linkCmd *cobra.Command) (link core.Linker, err error) {
	links := c.exec.Links()
	link = links[linkCmd.Name()]
	if link != nil {
		return
	}

	for _, alias := range linkCmd.Aliases {
		if link = links[alias]; link != nil {
			return
		}
	}

	err = fmt.Errorf("link not found: %q, aliases: %v", linkCmd.Name(), linkCmd.Aliases)
	return
}

func (c *cmdLinks) handle(chainedArgs [][]string, originalResult core.Result, parentOutputFlag string) (err error) {
	if c == nil {
		return
	}

	if len(chainedArgs) == 0 {
		return
	}

	args := chainedArgs[0]
	if len(args) == 0 {
		return
	}

	linkCmd, _, err := c.root.Find(args)
	if err != nil {
		logger().Debugw("link not found", "args", args, "error", err, "exec", c.exec, "links", c.exec.Links())
		return
	}

	link, err := c.findCommandLink(linkCmd)
	if err != nil {
		logger().Debugw("link not found", "args", args, "error", err, "exec", c.exec, "links", c.exec.Links(), "linkCmd", linkCmd)
		return
	}

	flags := c.cmdFlags[link.Name()] // safe: see addLinkCommand()

	sdk := c.sdk

	ctx := originalResult.Source().Context
	exec, err := link.CreateExecutor(originalResult)
	if err != nil {
		logger().Debugw("could not create link executor", "originalResult", originalResult, "error", err, "link", link)
		return
	}

	nextLinks := newCmdLinks(sdk, exec, fmt.Sprintf("%s ! %s", c.cmdPath, linkCmd.CommandPath()))

	linkCmd.RunE = func(cmd *cobra.Command, args []string) error {
		followingLinkArgs := chainedArgs[1:]
		if err, stop := nextLinks.check(followingLinkArgs); err != nil || stop {
			return err
		}

		config := sdk.Config()
		additionalParameters, additionalConfigs, err := flags.getValues(config, args)
		if err != nil {
			return err
		}

		printLinkExecutionTable(link.Name(), link.Description(), additionalParameters, additionalConfigs)
		result, err := handleExecutor(ctx, sdk, cmd, exec, additionalParameters, additionalConfigs)
		if err != nil {
			return err
		}

		return nextLinks.handle(followingLinkArgs, result, getOutputFlag(cmd))
	}

	setOutputFlag(c.root, parentOutputFlag)
	c.root.SetArgs(args)
	err = c.root.Execute()
	logger().Debugw("handled executor link", "exec", exec, "args", args, "error", err)
	return err
}

func (c *cmdLinks) addLinkCommand(link core.Linker) {
	linkFlags, _ := newCmdFlags(
		c.root,
		link.AdditionalParametersSchema(),
		link.AdditionalConfigsSchema(),
		nil,
	)

	linkName, aliases := getCommandNameAndAliases(link.Name())

	linkCmd := &cobra.Command{
		Use:     linkName,
		Aliases: aliases,
		Short:   link.Description(),
		GroupID: "links",
		Run:     func(cmd *cobra.Command, args []string) {}, // place holder
	}

	linkFlags.addFlags(linkCmd)
	c.cmdFlags[link.Name()] = linkFlags
	c.root.AddCommand(linkCmd)
}

func (c *cmdLinks) initCommands() {
	c.root = &cobra.Command{
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		SilenceErrors:     true,
		SilenceUsage:      true,
	}
	c.root.AddGroup(&cobra.Group{
		ID:    "links",
		Title: "Available links:",
	})

	addOutputFlag(c.root)
	addWaitTerminationFlag(c.root)
	addRetryUntilFlag(c.root)
	addBypassConfirmationFlag(c.root)

	// standard UsageTemplate replacing `{{.CommandPath}} [command]` with `%[1]s ! <link>`
	c.root.SetUsageTemplate(fmt.Sprintf(`Usage:{{if .Runnable}}
  %[1]s !{{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  %[1]s ! <link>{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "%[1]s ! <link> --help" for more information about a command.{{end}}
`, c.cmdPath))

	for _, link := range c.exec.Links() {
		c.addLinkCommand(link)
	}

	c.initHelp()
}

func (c *cmdLinks) initHelp() {
	c.root.InitDefaultHelpCmd()
	helpCmd, _, _ := c.root.Find([]string{"help"})
	helpCmd.Run = nil
	helpCmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		cmd, _, e := cmd.Root().Find(args)
		if cmd == nil || e != nil {
			err = fmt.Errorf("unknown help topic %#q", args)
		} else {
			if err = cmd.Help(); err == nil {
				err = schema_flags.ErrWantHelp
			}
		}
		return
	}
}

func formatMapTable(m map[string]any) string {
	t := table.NewWriter()
	t.AppendHeader(table.Row{"Name", "Value"})
	for k, v := range m {
		t.AppendRow(table.Row{k, v})
	}
	t.SortBy([]table.SortBy{{Number: 0}})
	t.SetStyle(table.StyleRounded)
	return t.Render()
}

func printLinkExecutionTable(name, description string, parameters, configs map[string]any) {
	t := table.NewWriter()
	t.AppendHeader(table.Row{"Executing link"})
	t.AppendRows([]table.Row{{"Name", name}, {"Description", description}})

	if len(parameters) > 0 {
		t.AppendRow(table.Row{"Parameters", formatMapTable(parameters)})
	}
	if len(configs) > 0 {
		t.AppendRow(table.Row{"Configs", formatMapTable(configs)})
	}

	t.SetStyle(table.StyleRounded)
	fmt.Println()
	fmt.Println(t.Render())
	fmt.Println()
}
