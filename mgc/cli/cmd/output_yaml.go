package cmd

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const defaultIndent = 4

type yamlOutputFormatter struct{}

func (*yamlOutputFormatter) Format(value any, options string) error {
	var indent int
	fmt.Sscanf(options, "%d", &indent)
	enc := yaml.NewEncoder(os.Stdout)
	if indent < 1 {
		indent = defaultIndent
	}
	enc.SetIndent(indent)
	return enc.Encode(value)
}

func init() {
	outputFormatters["yaml"] = &yamlOutputFormatter{}
}
