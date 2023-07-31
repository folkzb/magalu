package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const outputFlag = "cli.output"
const defaultFormatter = "json"

type OutputFormatter interface {
	Format(value any, options string) error
	// TODO: maybe add a way to explain the options? like a json schema
}

var outputFormatters = map[string]OutputFormatter{}

func getOutputFormats() []string {
	keys := make([]string, 0, len(outputFormatters))
	for k := range outputFormatters {
		keys = append(keys, k)
	}
	return keys
}

func addFormatterFlag(cmd *cobra.Command) {
	cmd.Root().PersistentFlags().StringP(
		outputFlag,
		"o",
		defaultFormatter,
		fmt.Sprintf("Output format. One of %s.", strings.Join(getOutputFormats(), "|")))
}

func getFormatter(cmd *cobra.Command) (formatter OutputFormatter, options string, err error) {
	spec := cmd.Root().PersistentFlags().Lookup(outputFlag).Value.String()
	parts := strings.SplitN(spec, "=", 2)
	name := parts[0]
	if len(parts) == 2 {
		options = parts[1]
	}

	if formatter, ok := outputFormatters[name]; ok {
		return formatter, options, nil
	}
	return nil, "", fmt.Errorf("unknown formatter %q", name)
}
