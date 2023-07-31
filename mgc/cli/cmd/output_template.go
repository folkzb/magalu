package cmd

import (
	"fmt"
	"os"

	"text/template"
)

type templateOutputFormatter struct{}

func (*templateOutputFormatter) Format(value any, options string) error {
	text := options
	tmpl, err := template.New("template").Parse(text)
	if err != nil {
		return fmt.Errorf("template output formatter: %w", err)
	}

	return tmpl.Execute(os.Stdout, value)
}

func init() {
	outputFormatters["template"] = &templateOutputFormatter{}
}
