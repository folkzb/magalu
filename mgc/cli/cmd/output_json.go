package cmd

import (
	"encoding/json"
	"os"
)

type jsonOutputFormatter struct{}

func (*jsonOutputFormatter) Format(value any, options string) error {
	enc := json.NewEncoder(os.Stdout)
	if options == "compact" {
		enc.SetIndent("", "")
	} else {
		enc.SetIndent("", " ")
	}
	return enc.Encode(value)
}

func init() {
	outputFormatters["json"] = &jsonOutputFormatter{}
}
