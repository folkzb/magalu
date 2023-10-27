package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	yamlReader "github.com/invopop/yaml"
	"github.com/spf13/cobra"
	yamlWriter "gopkg.in/yaml.v3"
	mgcSchemaPkg "magalu.cloud/core/schema"
)

var (
	outputYaml   bool
	outputIndent int
	outputFile   string
)

const (
	flagYaml      = "yaml"
	flagIndent    = "indent"
	flagFile      = "output"
	defaultIndent = 2
)

func getRootCmd() (rootCmd *cobra.Command) {
	rootCmd = &cobra.Command{
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"inputFile"},
		Use:       os.Args[0],
		Short:     "Reads a JSON Schema and prints out a simplified version of it",
		RunE:      run,
	}

	flags := rootCmd.Flags()
	flags.BoolVar(&outputYaml, flagYaml, false, "Output YAML instead of JSON")
	flags.IntVar(&outputIndent, flagIndent, 0, "Choose indentation level")
	flags.Lookup(flagIndent).NoOptDefVal = fmt.Sprint(defaultIndent)
	flags.StringVarP(&outputFile, flagFile, "o", "", "Output to the file instead of stdout")
	return
}

func formatYAML(schema *mgcSchemaPkg.Schema, writer io.Writer) error {
	tmp, err := json.Marshal(schema) // do this so MarshalJSON() is respected
	if err != nil {
		return nil
	}
	var v any
	err = json.Unmarshal(tmp, &v)
	if err != nil {
		return nil
	}

	enc := yamlWriter.NewEncoder(writer)
	if outputIndent < 1 {
		outputIndent = defaultIndent
	}
	enc.SetIndent(outputIndent)
	return enc.Encode(v)
}

func formatJSON(schema *mgcSchemaPkg.Schema, writer io.Writer) error {
	enc := json.NewEncoder(writer)
	if outputIndent < 0 {
		outputIndent = 0
	}
	enc.SetIndent("", strings.Repeat(" ", outputIndent))
	return enc.Encode(schema)
}

func format(schema *mgcSchemaPkg.Schema, writer io.Writer) error {
	if outputYaml {
		return formatYAML(schema, writer)
	}
	return formatJSON(schema, writer)
}

func formatString(schema *mgcSchemaPkg.Schema) string {
	writer := &strings.Builder{}
	_ = format(schema, writer)
	return writer.String()
}

func run(cmd *cobra.Command, args []string) (err error) {
	inputFile := args[0]
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return err
	}

	output := os.Stdout
	if outputFile != "" && outputFile != "-" {
		output, err = os.Create(outputFile)
		if err != nil {
			return err
		}
	}

	var schema *mgcSchemaPkg.Schema
	err = yamlReader.Unmarshal(data, &schema)
	if err != nil {
		return fmt.Errorf("cannot unmarshal %q: %w", inputFile, err)
	}

	simplified, err := mgcSchemaPkg.SimplifySchema(schema)
	if err != nil {
		return fmt.Errorf("cannot simplify %q: %w\n%s", inputFile, err, formatString(schema))
	}

	return format(simplified, output)
}

func main() {
	rootCmd := getRootCmd()
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
