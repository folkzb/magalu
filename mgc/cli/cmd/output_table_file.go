package cmd

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type tableFileOutputFormatter struct{}

func (*tableFileOutputFormatter) Format(value any, options string) error {
	filename := options
	file, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("table-file output formatter: %w", err)
	}

	tableOpts := &tableOptions{}
	err = yaml.Unmarshal(file, tableOpts)

	if err != nil {
		return err
	}

	return formatTableWithOptions(value, tableOpts)
}

func init() {
	outputFormatters["table-file"] = &tableFileOutputFormatter{}
}
