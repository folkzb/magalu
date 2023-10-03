package cmd

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
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
