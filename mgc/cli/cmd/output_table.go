package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/scanner"

	"github.com/PaesslerAG/jsonpath"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

type tableOutputFormatter struct{}

type alignString text.Align

func (a *alignString) UnmarshalYAML(node *yaml.Node) error {
	switch strings.ToLower(node.Value) {
	case "default", "aligndefault":
		*a = alignString(text.AlignDefault)
	case "left", "alignleft":
		*a = alignString(text.AlignLeft)
	case "center", "aligncenter":
		*a = alignString(text.AlignCenter)
	case "justify", "alignjustify":
		*a = alignString(text.AlignJustify)
	case "right", "alignright":
		*a = alignString(text.AlignRight)
	default:
		*a = alignString(text.AlignDefault)
	}

	return nil
}

type valignString text.VAlign

func (a *valignString) UnmarshalYAML(node *yaml.Node) error {
	switch strings.ToLower(node.Value) {
	case "default", "aligndefault":
		*a = valignString(text.VAlignDefault)
	case "top", "aligntop":
		*a = valignString(text.VAlignTop)
	case "middle", "alignmiddle":
		*a = valignString(text.VAlignMiddle)
	case "bottom", "alignbottom":
		*a = valignString(text.VAlignBottom)
	default:
		*a = valignString(text.VAlignDefault)
	}

	return nil
}

type colorString text.Color

func (c *colorString) UnmarshalYAML(node *yaml.Node) error {
	switch strings.ToLower(node.Value) {
	case "reset":
		*c = colorString(text.Reset)
	// Styling
	case "bold":
		*c = colorString(text.Bold)
	case "faint":
		*c = colorString(text.Faint)
	case "italic":
		*c = colorString(text.Italic)
	case "underline":
		*c = colorString(text.Underline)
	case "reversevideo":
		*c = colorString(text.ReverseVideo)
	case "concealed":
		*c = colorString(text.Concealed)
	case "crossedout":
		*c = colorString(text.CrossedOut)
	// Blinking
	case "blinkslow":
		*c = colorString(text.BlinkSlow)
	case "blinkrapid":
		*c = colorString(text.BlinkRapid)
	// Foreground colors
	case "black", "fgblack":
		*c = colorString(text.FgBlack)
	case "red", "fgred":
		*c = colorString(text.FgRed)
	case "green", "fggreen":
		*c = colorString(text.FgGreen)
	case "yellow", "fgyellow":
		*c = colorString(text.FgYellow)
	case "blue", "fgblue":
		*c = colorString(text.FgBlue)
	case "magenta", "fgmagenta":
		*c = colorString(text.FgMagenta)
	case "cyan", "fgcyan":
		*c = colorString(text.FgCyan)
	case "white", "fgwhite":
		*c = colorString(text.FgWhite)
	// Foreground high intensity colors
	case "fghiblack":
		*c = colorString(text.FgHiBlack)
	case "fghired":
		*c = colorString(text.FgHiRed)
	case "fghigreen":
		*c = colorString(text.FgHiGreen)
	case "fghiyellow":
		*c = colorString(text.FgHiYellow)
	case "fghiblue":
		*c = colorString(text.FgHiBlue)
	case "fghimagenta":
		*c = colorString(text.FgHiMagenta)
	case "fghicyan":
		*c = colorString(text.FgHiCyan)
	case "fghiwhite":
		*c = colorString(text.FgHiWhite)
	// Background colors
	case "bgblack":
		*c = colorString(text.BgBlack)
	case "bgred":
		*c = colorString(text.BgRed)
	case "bggreen":
		*c = colorString(text.BgGreen)
	case "bgyellow":
		*c = colorString(text.BgYellow)
	case "bgblue":
		*c = colorString(text.BgBlue)
	case "bgmagenta":
		*c = colorString(text.BgMagenta)
	case "bgcyan":
		*c = colorString(text.BgCyan)
	case "bgwhite":
		*c = colorString(text.BgWhite)
	// Background high intensity colors
	case "bghiblack":
		*c = colorString(text.BgHiBlack)
	case "bghired":
		*c = colorString(text.BgHiRed)
	case "bghigreen":
		*c = colorString(text.BgHiGreen)
	case "bghiyellow":
		*c = colorString(text.BgHiYellow)
	case "bghiblue":
		*c = colorString(text.BgHiBlue)
	case "bghimagenta":
		*c = colorString(text.BgHiMagenta)
	case "bghicyan":
		*c = colorString(text.BgHiCyan)
	case "bghiwhite":
		*c = colorString(text.BgHiWhite)
	default:
		return fmt.Errorf("unknown column color value: %s", node.Value)
	}

	return nil
}

type colorStrings []colorString

func (c *colorStrings) toColors() text.Colors {
	result := make(text.Colors, len(*c))
	for i, color := range *c {
		result[i] = text.Color(color)
	}
	return result
}

type columnConfig struct {
	Align        alignString  `yaml:"align"`
	AlignFooter  alignString  `yaml:"alignFooter"`
	AlignHeader  alignString  `yaml:"alignHeader"`
	AutoMerge    bool         `yaml:"autoMerge"`
	Colors       colorStrings `yaml:"colors"`
	ColorsFooter colorStrings `yaml:"colorsFooter"`
	ColorsHeader colorStrings `yaml:"colorsHeader"`
	Hidden       bool         `yaml:"hidden"`
	VAlign       valignString `yaml:"valign"`
	VAlignFooter valignString `yaml:"valignFooter"`
	VAlignHeader valignString `yaml:"valignHeader"`
	WidthMax     int          `yaml:"widthMax"`
	WidthMin     int          `yaml:"widthMin"`
}

type column struct {
	Name     string
	JSONPath string
	Config   *columnConfig
}

func (c *column) tableColumnConfig() *table.ColumnConfig {
	t := &table.ColumnConfig{
		Name:         c.Name,
		Align:        text.Align(c.Config.Align),
		AlignFooter:  text.Align(c.Config.AlignFooter),
		AlignHeader:  text.Align(c.Config.AlignHeader),
		AutoMerge:    c.Config.AutoMerge,
		Colors:       c.Config.Colors.toColors(),
		ColorsFooter: c.Config.ColorsFooter.toColors(),
		ColorsHeader: c.Config.ColorsHeader.toColors(),
		Hidden:       c.Config.Hidden,
		VAlign:       text.VAlign(c.Config.VAlign),
		VAlignFooter: text.VAlign(c.Config.VAlignFooter),
		VAlignHeader: text.VAlign(c.Config.VAlignHeader),
		WidthMax:     c.Config.WidthMax,
		WidthMin:     c.Config.WidthMin,
	}
	return t
}

type tableRenderFormat string

const (
	Ascii    tableRenderFormat = "ascii"
	HTML     tableRenderFormat = "html"
	Markdown tableRenderFormat = "markdown"
	CSV      tableRenderFormat = "csv"
)

type tableStyleString table.Style

func (t *tableStyleString) UnmarshalYAML(node *yaml.Node) error {
	switch strings.ToLower(node.Value) {
	case "bold", "stylebold":
		*t = tableStyleString(table.StyleBold)
	case "coloredblackonbluewhite", "stylecoloredblackonbluewhite":
		*t = tableStyleString(table.StyleColoredBlackOnBlueWhite)
	case "coloredblackoncyanwhite", "stylecoloredblackoncyanwhite":
		*t = tableStyleString(table.StyleColoredBlackOnCyanWhite)
	case "coloredblackongreenwhite", "stylecoloredblackongreenwhite":
		*t = tableStyleString(table.StyleColoredBlackOnGreenWhite)
	case "coloredblackonmagentawhite", "stylecoloredblackonmagentawhite":
		*t = tableStyleString(table.StyleColoredBlackOnMagentaWhite)
	case "coloredblackonredwhite", "stylecoloredblackonredwhite":
		*t = tableStyleString(table.StyleColoredBlackOnRedWhite)
	case "coloredblackonyellowwhite", "stylecoloredblackonyellowwhite":
		*t = tableStyleString(table.StyleColoredBlackOnYellowWhite)
	case "coloredbright", "stylecoloredbright":
		*t = tableStyleString(table.StyleColoredBright)
	case "coloredcyanwhiteonblack", "stylecoloredcyanwhiteonblack":
		*t = tableStyleString(table.StyleColoredCyanWhiteOnBlack)
	case "coloreddark", "stylecoloreddark":
		*t = tableStyleString(table.StyleColoredDark)
	case "coloredgreenwhiteonblack", "stylecoloredgreenwhiteonblack":
		*t = tableStyleString(table.StyleColoredGreenWhiteOnBlack)
	case "coloredmagentawhiteonblack", "stylecoloredmagentawhiteonblack":
		*t = tableStyleString(table.StyleColoredMagentaWhiteOnBlack)
	case "coloredredwhiteonblack", "stylecoloredredwhiteonblack":
		*t = tableStyleString(table.StyleColoredRedWhiteOnBlack)
	case "coloredyellowwhiteonblack", "stylecoloredyellowwhiteonblack":
		*t = tableStyleString(table.StyleColoredYellowWhiteOnBlack)
	case "default", "styledefault":
		*t = tableStyleString(table.StyleDefault)
	case "double", "styledouble":
		*t = tableStyleString(table.StyleDouble)
	case "light", "stylelight":
		*t = tableStyleString(table.StyleLight)
	case "rounded", "stylerounded":
		*t = tableStyleString(table.StyleRounded)
	default:
		*t = tableStyleString(table.StyleDefault)
	}

	return nil
}

type tableOptions struct {
	Columns   []*column
	RowLength int
	Format    tableRenderFormat
	Style     *tableStyleString
}

func concreteKind(v reflect.Value) reflect.Kind {
	kind := v.Kind()
	if kind == reflect.Interface || kind == reflect.Pointer {
		kind = v.Elem().Kind()
	}
	return kind
}

func columnsFromPointerOrInterface(v reflect.Value, prefix string) ([]*column, error) {
	if v.IsNil() {
		return []*column{{Name: "RESULT", JSONPath: prefix}}, nil
	}

	return columnsFromAny(v.Elem().Interface(), prefix)
}

func columnsFromStruct(v reflect.Value, prefix string) ([]*column, error) {
	t := v.Type()
	fieldCount := t.NumField()

	result := make([]*column, fieldCount)
	for i := 0; i < fieldCount; i++ {
		field := t.Field(i)
		if field.IsExported() {
			result[i] = &column{
				Name:     strings.ToUpper(field.Name),
				JSONPath: fmt.Sprintf("%s[%q]", prefix, field.Name),
			}
		}
	}

	return result, nil
}

func columnsFromArrayOrSlice(v reflect.Value, prefix string) ([]*column, error) {
	if v.Len() == 0 {
		return []*column{{Name: "RESULT", JSONPath: prefix}}, nil
	}

	subVal := v.Index(0)
	return columnsFromAny(subVal.Interface(), prefix+"[*]")
}

func columnsFromMap(v reflect.Value, prefix string) ([]*column, error) {
	length := v.Len()
	keys := v.MapKeys()

	// Check for prefix because we don't want the nesting to be deep, only one level
	if length == 1 && prefix == "$" {
		key := keys[0]
		if key.Kind() == reflect.String {
			subVal := v.MapIndex(key)
			subKind := concreteKind(subVal)

			switch subKind {
			case reflect.Array, reflect.Slice:
				// {
				//     "resultList": [
				//         { "field1": 1, "field2", 2 },
				//         { "field1": 3, "field2", 4 },
				//     ]
				// }
				return columnsFromAny(subVal.Interface(), fmt.Sprintf("%s[%q]", prefix, key))
			case reflect.Map:
				// {
				//     "result": {
				//         "field1": 1,
				//         "field2": 2,
				//     }
				// }
				return columnsFromAny(subVal.Interface(), fmt.Sprintf("%s[%q]", prefix, key))
			}
		}
	}
	// {
	//     "field1": 1,
	//     "field2": 2,
	// }
	result := make([]*column, len(keys))
	for i, key := range keys {
		keyStr := key.String()
		result[i] = &column{
			Name:     strings.ToUpper(keyStr),
			JSONPath: fmt.Sprintf("%s[%q]", prefix, keyStr),
		}
	}
	slices.SortFunc(result, func(l *column, r *column) int {
		return strings.Compare(l.Name, r.Name)
	})
	return result, nil
}

func columnsFromAny(val any, prefix string) ([]*column, error) {
	if prefix == "" {
		prefix = "$"
	}

	v := reflect.ValueOf(val)
	if !v.IsValid() {
		return []*column{{Name: "RESULT", JSONPath: prefix}}, nil
	}

	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		return columnsFromPointerOrInterface(v, prefix)
	case reflect.Struct:
		return columnsFromStruct(v, prefix)
	case reflect.Array, reflect.Slice:
		return columnsFromArrayOrSlice(v, prefix)
	case reflect.Map:
		return columnsFromMap(v, prefix)
	default:
		return []*column{{Name: "RESULT", JSONPath: prefix}}, nil
	}
}

func splitUnquoted(str string, separator string) (result []string, err error) {
	cur := ""
	result = []string{}
	var s scanner.Scanner
	s.Init(strings.NewReader(str))
	s.Mode |= scanner.ScanStrings
	s.Error = func(s *scanner.Scanner, msg string) {} // function does nothing so errors are not printed
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		txt := s.TokenText()
		if txt == "" {
			continue
		}
		switch txt[0] {
		case '"', '\'', '`':
			txt, err = strconv.Unquote(txt)
			if err != nil {
				return
			}
		default:
		}
		if txt == separator {
			result = append(result, cur)
			cur = ""
		} else {
			cur += txt
		}
	}
	if cur != "" {
		result = append(result, cur)
	}
	return
}

func columnsFromString(str string) ([]*column, error) {
	colStrings, err := splitUnquoted(str, ",")
	if err != nil {
		return nil, err
	}
	result := make([]*column, 0)
	for _, colString := range colStrings {
		if colString == "" {
			continue
		}
		parts := strings.SplitN(colString, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("wrong table column format: %s", colString)
		}

		name := parts[0]
		jsonPath := parts[1]

		result = append(result, &column{name, jsonPath, nil})
	}
	return result, nil
}

func configureWriter(w table.Writer, options *tableOptions) {
	if options.RowLength != 0 {
		w.SetAllowedRowLength(options.RowLength)
	} else {
		maxLength := os.Getenv("MAX_ROW_LENGTH")
		num, err := strconv.Atoi(maxLength)
		if err != nil || num == 0 {
			num = 150
		}
		w.SetAllowedRowLength(num)
	}

	if options.Style != nil {
		w.SetStyle(table.Style(*options.Style))
	}
}

func formatTableWithOptions(val any, options *tableOptions) error {
	columnCount := len(options.Columns)
	headers := make([]any, columnCount)
	for i, col := range options.Columns {
		headers[i] = col.Name
	}

	writer := table.NewWriter()
	writer.AppendHeader(headers)
	configureWriter(writer, options)

	rows := []table.Row{}
	configs := []table.ColumnConfig{}
	handleVal := func(rowIdx int, colIdx int, value any) error {
		if rowIdx >= len(rows) {
			rows = append(rows, make(table.Row, columnCount))
		}

		var sanitized any
		switch value.(type) {
		case bool, *bool, int, *int, int8, *int8, int16, *int16, int32, *int32, int64, *int64, uint, *uint, uint8, *uint8, uint16, *uint16, uint32, *uint32, uint64, *uint64, float32, *float32, string, *string:
			sanitized = value
		default:
			// Marshall the value for easier reading when printing to console, even if inside of a table
			marshalled, err := json.Marshal(value)
			if err != nil {
				return err
			}
			sanitized = string(marshalled)
		}

		rows[rowIdx][colIdx] = sanitized
		return nil
	}

	for colIdx, col := range options.Columns {
		result, err := jsonpath.Get(col.JSONPath, val)

		if err != nil {
			return err
		}

		if arr, ok := result.([]any); ok {
			for rowIdx, v := range arr {
				err := handleVal(rowIdx, colIdx, v)
				if err != nil {
					return err
				}
			}
		} else {
			err := handleVal(0, colIdx, result)
			if err != nil {
				return err
			}
		}

		if col.Config != nil {
			configs = append(configs, *col.tableColumnConfig())
		}
	}

	if len(configs) != 0 {
		writer.SetColumnConfigs(configs)
	}

	writer.AppendRows(rows)
	fmt.Println(renderWriterWithFormat(writer, options.Format))
	return nil
}

func renderWriterWithFormat(writer table.Writer, format tableRenderFormat) string {
	switch format {
	case Ascii:
		return writer.Render()
	case HTML:
		return writer.RenderHTML()
	case CSV:
		return writer.RenderCSV()
	case Markdown:
		return writer.RenderMarkdown()
	}

	return writer.Render()
}

func (f *tableOutputFormatter) Format(val any, options string) error {
	var columns []*column
	if options != "" {
		fromString, err := columnsFromString(options)
		if err != nil {
			return err
		}
		columns = fromString
	} else {
		fromAny, err := columnsFromAny(val, "$")
		if err != nil {
			return err
		}
		columns = fromAny
	}

	tableOptions := &tableOptions{Columns: columns}

	return formatTableWithOptions(val, tableOptions)
}

func init() {
	outputFormatters["table"] = &tableOutputFormatter{}
}
