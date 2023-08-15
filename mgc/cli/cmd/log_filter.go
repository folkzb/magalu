package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

const logFilterFlag = "cli.log"

func addLogFilterFlag(cmd *cobra.Command) {
	cmd.Root().PersistentFlags().StringP(
		logFilterFlag,
		"l",
		"",
		"Use \"*:*\" to show all logs. See more details about the filter syntax at https://github.com/moul/zapfilter",
	)
}

func getLogFilterFlag(cmd *cobra.Command) string {
	_ = cmd.ParseFlags(os.Args[1:])
	return cmd.Root().PersistentFlags().Lookup(logFilterFlag).Value.String()
}
