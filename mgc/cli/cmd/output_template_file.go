package cmd

import (
	"fmt"
	"os"

	"text/template"
)

type templateFileOutputFormatter struct{}

func (*templateFileOutputFormatter) Format(value any, options string) error {
	filename := options
	text, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("template-file output formatter: %w", err)
	}
	tmpl, err := template.New(filename).Parse(string(text))
	if err != nil {
		return fmt.Errorf("template-file output formatter: %w", err)
	}

	return tmpl.Execute(os.Stdout, value)
}

func (*templateFileOutputFormatter) Description() string {
	return `Same as template, but reads the value from the given file: "template-file=path-to-file-with-template"`
}

func init() {
	outputFormatters["template-file"] = &templateFileOutputFormatter{}
}
