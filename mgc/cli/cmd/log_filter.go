package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

const logFilterFlag = "cli.log"

func addLogFilterFlag(cmd *cobra.Command, def string) {
	if def == "" {
		def = "warn+:*"
	}
	cmd.Root().PersistentFlags().StringP(
		logFilterFlag,
		"l",
		def,
		"Format is \"levels:namespaces\". Use \"info+:*\" to show info for all levels, use \"*:*\" to show all logs. See more details about the filter syntax at https://github.com/moul/zapfilter",
	)
}

func getLogFilterFlag(cmd *cobra.Command) string {
	_ = cmd.ParseFlags(os.Args[1:])
	return cmd.Root().PersistentFlags().Lookup(logFilterFlag).Value.String()
}
