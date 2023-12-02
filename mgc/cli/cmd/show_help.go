package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
	"magalu.cloud/cli/cmd/schema_flags"
	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
)

func showFormatHelp() {
	writer := table.NewWriter()
	writer.AppendHeader(table.Row{"Formatter", "Description"})
	maxLen := 0
	for k, f := range outputFormatters {
		if maxLen < len(k) {
			maxLen = len(k)
		}
		writer.AppendRow(table.Row{k, f.Description()})
	}

	termColumns := getTermColumns()
	tablePadding := 4 // 2 columns x 2 spaces per column
	writer.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, Align: text.AlignLeft, VAlign: text.VAlignTop, WidthMax: maxLen},
		{Number: 2, Align: text.AlignLeft, VAlign: text.VAlignTop, WidthMax: termColumns - maxLen - tablePadding},
	})
	style := table.StyleDefault
	style.Options = table.OptionsNoBordersAndSeparators
	style.Options.SeparateHeader = true

	writer.SetStyle(style)
	writer.SortBy([]table.SortBy{{Name: "Formatter", Mode: table.Asc}})

	fmt.Println("For plain data types, the following values are accepted:")

	fmt.Println(writer.Render())

	fmt.Println("\nFor streams, use the file name to save to or '-' to write to stdout (default).")
}

func showHelpForError(cmd *cobra.Command, args []string, err error) error {
	switch {
	case err == schema_flags.ErrWantHelp:
		return nil

	case errors.As(err, new(core.UsageError)):
		// we can't call UsageString() on the root, we need to find the actual leaf command that failed:
		subCmd, _, _ := cmd.Find(args)
		cmd.PrintErrln(subCmd.UsageString())
	default:
		break
	}

	return err
}

func textIndent(s, prefix string) string {
	return prefix + strings.Join(strings.Split(s, "\n"), "\n"+prefix)
}

func textReflow(s, prefix string, columns int) string {
	if columns > 0 {
		s = text.WrapText(s, columns)
	}

	if prefix != "" {
		s = textIndent(s, prefix)
	}

	return s
}

func cleanJSONSchemaCOW(cowSchema *mgcSchemaPkg.COWSchema) {
	_ = cowSchema.SetExample(nil)
	_ = cowSchema.SetExtensions(nil)

	if cowSchema.Items() != nil {
		cleanJSONSchemaRefCOW(cowSchema.ItemsCOW())
	}

	if len(cowSchema.Properties()) > 0 {
		cowSchema.PropertiesCOW().ForEachCOW(func(_ string, cow *mgcSchemaPkg.COWSchemaRef) (run bool) {
			cleanJSONSchemaRefCOW(cow)
			return true
		})
	}

	if len(cowSchema.AllOf()) > 0 {
		cowSchema.AllOfCOW().ForEachCOW(func(_ int, cow *mgcSchemaPkg.COWSchemaRef) (run bool) {
			cleanJSONSchemaRefCOW(cow)
			return true
		})
	}

	if len(cowSchema.AnyOf()) > 0 {
		cowSchema.AnyOfCOW().ForEachCOW(func(_ int, cow *mgcSchemaPkg.COWSchemaRef) (run bool) {
			cleanJSONSchemaRefCOW(cow)
			return true
		})
	}

	if len(cowSchema.OneOf()) > 0 {
		cowSchema.OneOfCOW().ForEachCOW(func(_ int, cow *mgcSchemaPkg.COWSchemaRef) (run bool) {
			cleanJSONSchemaRefCOW(cow)
			return true
		})
	}
}

func cleanJSONSchemaRefCOW(cowSchemaRef *mgcSchemaPkg.COWSchemaRef) {
	if cowSchemaRef.Value() == nil {
		return
	}
	cleanJSONSchemaCOW(cowSchemaRef.ValueCOW())
}

// Removes examples, extensions and whatever else may not be relevant to the user
func getCleanJSONSchema(schema *core.Schema) *core.Schema {
	cowSchema := mgcSchemaPkg.NewCOWSchema(schema)
	cowSchema.SetDescription("") // the root description was already displayed
	cleanJSONSchemaCOW(cowSchema)
	return cowSchema.Peek()
}

func getExample(schema, container *core.Schema, propName string) (example any) {
	example = schema.Example
	if example != nil {
		return
	}

	if schema.Type == "array" && schema.Items != nil && schema.Items.Value != nil && schema.Items.Value.Example != nil {
		return []any{schema.Items.Value.Example}
	}

	if container.Example == nil {
		return
	}

	if containerExample, ok := container.Example.(map[string]any); ok {
		example = containerExample[propName]
		if example != nil {
			return
		}
	}

	return
}

func getExampleFormattedValue(schema, container *core.Schema, propName string) (value string) {
	example := getExample(schema, container, propName)
	if example == nil {
		return
	}

	data, err := json.Marshal(example)
	if err != nil {
		return
	}

	switch schema.Type {
	case "integer", "number", "boolean":
		return string(data)

	case "string":
		value = string(data)
		if value == schema_flags.ValueHelpIsRequired {
			value = schema_flags.ValueVerbatimStringPrefix + value
		} else if strings.HasPrefix(value, schema_flags.ValueVerbatimStringPrefix) || strings.Contains(value, "$") {
			value = fmt.Sprintf("'%s'", data) // keep quotes and wrap in single, so shell doesn't replace variables
		}
		return

	default:
		return fmt.Sprintf("'%s'", data)
	}
}

func getFlagHelpUsageLine(f *flag.Flag) (s string) {
	s = "--" + f.Name
	t := f.Value.Type()

	if f.NoOptDefVal != "" {
		s += fmt.Sprintf("[=%s] (if no explicit value is given, uses: %s)", t, f.NoOptDefVal)
	} else {
		s += "=" + t
	}

	return
}

func getFlagDescription(f *flag.Flag) (description string) {
	if fv, ok := f.Value.(schema_flags.SchemaFlagValue); ok {
		description = fv.Desc().Description()
	} else if f.Usage != "" {
		description = f.Usage
	}

	return description
}

func showFlagHelp(f *flag.Flag) {
	const (
		indentPrefix = "    "
	)
	columns := getTermColumns()
	reflowColumns := columns - len(indentPrefix)
	var output string

	reflow := func(s string) string {
		return textReflow(s, indentPrefix, reflowColumns)
	}

	addSection := func(title string, rest ...string) {
		if output != "" {
			output += "\n"
		}

		output += fmt.Sprintf("%s:", title)
		if len(rest) > 0 {
			for _, s := range rest {
				output += " " + s
			}
		}
		output += "\n"
	}
	addSectionBodyText := func(body string) {
		output += reflow(body) + "\n"
	}
	addSectionBodyRaw := func(body string) {
		output += indentPrefix + body + "\n"
	}

	addSection("Flag usage")
	addSectionBodyRaw(getFlagHelpUsageLine(f))

	if description := getFlagDescription(f); description != "" {
		addSection("Description")
		addSectionBodyText(description)
	}

	if f.DefValue != "" {
		addSection("Default value", f.DefValue)
	}

	if fv, ok := f.Value.(schema_flags.SchemaFlagValue); ok {
		desc := fv.Desc()
		cleanSchema := getCleanJSONSchema(desc.Schema)
		if data, err := json.MarshalIndent(cleanSchema, indentPrefix, "  "); err == nil {
			schema := strings.Trim(string(data), "\t\n\r ")
			if schema != "{}" {
				addSection("JSON Schema")
				addSectionBodyRaw(schema)
			}
		}

		if example := getExampleFormattedValue(desc.Schema, desc.Container, desc.PropName); example != "" {
			addSection("Example")
			addSectionBodyRaw(fmt.Sprintf("--%s=%s", f.Name, example))
		}
	}

	fmt.Println(output)
}
