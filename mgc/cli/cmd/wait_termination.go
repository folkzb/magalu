package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	waitTerminationFlag string = "cli.wait-termination"
)

func addWaitTerminationFlag(cmd *cobra.Command) {
	cmd.Root().PersistentFlags().StringP(
		waitTerminationFlag,
		"w",
		"",
		"Conditions for request polling. Format is \"retries,interval\" or \"retries\". For values <= 0 the CLI will use the action default value",
	)

	f := cmd.Root().PersistentFlags().Lookup(waitTerminationFlag)
	f.NoOptDefVal = "0,0"
}

func getWaitTerminationFlag(cmd *cobra.Command) (int, time.Duration, error) {
	wt, err := cmd.Root().PersistentFlags().GetString(waitTerminationFlag)
	if err != nil {
		return 0, 0, err
	}

	return parseWaitTermination(wt)
}

func parseWaitTermination(wt string) (maxRetries int, interval time.Duration, err error) {
	maxRetries = 0
	interval = 0

	if wt == "" {
		err = fmt.Errorf("value for request polling should be in the form retries,interval")
		return
	}

	p := strings.SplitN(wt, ",", 2)

	if _, err = fmt.Sscanf(p[0], "%d", &maxRetries); err != nil {
		err = fmt.Errorf("value for request polling should be in the form retries,interval. Failed to parse retries: %w", err)
		return
	}

	if len(p) > 1 {
		interval, err = time.ParseDuration(p[1])
		if err != nil {
			err = fmt.Errorf("value for request polling should be in the form retries,interval. Failed to parse interval: %w", err)
			return
		}
	}

	return
}
