package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

const hideProgressFlag = "cli.no-progress"

func addHideProgressFlag(cmd *cobra.Command) {
	cmd.Root().PersistentFlags().BoolP(
		hideProgressFlag,
		"p",
		false,
		"Hides progress bars if set",
	)
	f := cmd.Root().PersistentFlags().Lookup(hideProgressFlag)
	f.NoOptDefVal = "true"
}

func getHideProgressFlag(cmd *cobra.Command) bool {
	_ = cmd.ParseFlags(os.Args[1:])
	value, err := cmd.Root().PersistentFlags().GetBool(hideProgressFlag)
	if err != nil {
		return false
	}
	return value
}
