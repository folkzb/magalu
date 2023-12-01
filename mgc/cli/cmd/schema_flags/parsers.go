package schema_flags

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode"

	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
)

const (
	ValueLoadJSONFromFilePrefix     = "@"
	ValueLoadVerbatimFromFilePrefix = "%"
	ValueVerbatimStringPrefix       = "#"
)

// handles special cases, in order:
//  1. "": empty value is returned. No error.
//  2. "@filename": load JSON from file.
//     Returns error if file was not found or it's not a valid JSON for type "T".
//  3. try to JSON parse as value type "T"
func parseJSONFlagValue[T any](rawValue string) (value T, err error) {
	switch {
	case rawValue == "":
		return

	case strings.HasPrefix(rawValue, ValueLoadJSONFromFilePrefix):
		return loadJSONFromFile[T](rawValue[1:])

	default:
		err = json.Unmarshal([]byte(rawValue), &value)
		return
	}
}

// handles special cases targeted at strings, in addition to parseJSON(), in order:
//  1. "%filename": load raw string from filename (no trim or parsing is done).
//     Returns error if file was not found.
//  2. "#string": use the rest of the string verbatim (no trim or parsing is done). No error.
//  3. parseJSON(), if errors then use the rawValue instead. No error
//
// To pass a string with leading-and-trailing quotes (`"value"`)
// one must either provide a version with escaped quotes (`"\"value\""`)
// or use the verbatim prefix (`#"value"`)
func parseStringFlagValue(rawValue string) (value string, err error) {
	switch {
	case strings.HasPrefix(rawValue, ValueLoadVerbatimFromFilePrefix):
		return loadVerbatimFromFile(rawValue[1:])

	case strings.HasPrefix(rawValue, ValueVerbatimStringPrefix):
		return rawValue[1:], nil

	default:
		value, err = parseJSONFlagValue[string](rawValue)
		if err != nil {
			value = rawValue
			err = nil
		}
		return
	}
}

func parseBoolFlagValue(rawValue string) (value bool, err error) {
	if rawValue == "" {
		return // default to false
	}
	return strconv.ParseBool(rawValue) // mimics pflag's boolValue
}

func isWhiteSpace(c rune) bool {
	return unicode.IsSpace(c)
}

func isCSVDelimiter(c rune) bool {
	return c == ',' || c == ';'
}

func isCSVDelimiterOrWhiteSpace(c rune) bool {
	return isCSVDelimiter(c) || isWhiteSpace(c)
}

func isObjectKeyDelimiter(c rune) bool {
	return c == ':' || c == '='
}

func skipLeadingMatches(s string, isMatch func(rune) bool) (end int) {
	for i, c := range s {
		if !isMatch(c) {
			return i
		}
	}

	return len(s)
}

func skipWhiteSpaces(s string) (end int) {
	return skipLeadingMatches(s, isWhiteSpace)
}

func skipCSVDelimitersOrWhiteSpaces(s string) (end int) {
	return skipLeadingMatches(s, isCSVDelimiterOrWhiteSpace)
}

func parseNonWhitespaceCSVItem(s string) (item string, end int) {
	for i, c := range s {
		if isCSVDelimiterOrWhiteSpace(c) {
			end = i + 1
			return s[:i], end
		}
	}

	return s, len(s)
}

func parseNumberItem(s string) (value any, end int, err error) {
	text, end := parseNonWhitespaceCSVItem(s)
	value, err = strconv.ParseFloat(text, 64)
	return
}

func parseIntegerItem(s string) (value any, end int, err error) {
	text, end := parseNonWhitespaceCSVItem(s)
	i, err := strconv.ParseInt(text, 10, 64)
	if math.MinInt <= i && i <= math.MaxInt {
		value = int(i)
	} else {
		value = i
	}
	return
}

func parseBooleanItem(s string) (value any, end int, err error) {
	text, end := parseNonWhitespaceCSVItem(s)
	value, err = strconv.ParseBool(text)
	return
}

func parseQuotedStringItem(s string) (value string, end int, err error) {
	var quote rune
	escaped := false

	for i, c := range s {
		if i == 0 {
			quote = c
			continue
		}

		if escaped {
			escaped = false
		} else if c == '\\' {
			escaped = true
			continue
		} else if c == quote {
			end = i + 1
			end += skipCSVDelimitersOrWhiteSpaces(s[end:])
			return
		}

		value += string(c)
	}

	return value, len(s), fmt.Errorf("missing end quote char %c: %s", quote, s)
}

func parseString(s string, isDelimiter func(rune) bool) (value string, end int, err error) {
	start := skipWhiteSpaces(s)
	remaining := s[start:]
	if len(remaining) > 1 && (remaining[0] == '"' || remaining[0] == '\'') {
		value, end, err = parseQuotedStringItem(remaining)
		end += start
		return
	}

	if remaining == "" {
		end = start
		return
	}

	lastNonWhitespace := 0
	for i, c := range remaining {
		if isDelimiter(c) {
			end = start + i + 1
			value = remaining[:lastNonWhitespace+1]
			return
		} else if !isWhiteSpace(c) {
			lastNonWhitespace = i
		}
	}

	return remaining[:lastNonWhitespace+1], len(s), nil
}

func parseStringItem(s string) (value any, end int, err error) {
	return parseString(s, isCSVDelimiter)
}

func parseAnyItem(s string) (value any, end int, err error) {
	start := skipWhiteSpaces(s)
	end = start
	if start >= len(s) {
		return
	}

	if err = json.Unmarshal([]byte(s[start:]), &value); err == nil {
		end = len(s)
		return
	}

	var syntaxError = new(json.SyntaxError)
	if !errors.As(err, &syntaxError) {
		return
	}

	errorAt := int(syntaxError.Offset - 1)
	if errorAt <= 0 {
		return parseStringItem(s)
	}

	var c rune
	if n, _ := fmt.Sscanf(syntaxError.Error(), "invalid character '%c' after top-level value", &c); n != 1 || !isCSVDelimiter(c) {
		return
	}

	err = json.Unmarshal([]byte(s[start:errorAt]), &value)
	end = start + errorAt + 1
	return
}

func getItemTypeParser(schema *core.Schema) (itemParser func(s string) (any, int, error)) {
	var itemType string
	if schema != nil {
		itemType = schema.Type
	}

	switch itemType {
	case "number":
		return parseNumberItem
	case "integer":
		return parseIntegerItem
	case "boolean":
		return parseBooleanItem
	case "string":
		return parseStringItem
	default:
		return parseAnyItem
	}
}

func parseArrayCSV(itemsSchema *core.Schema, rawValue string) (value []any, err error) {
	itemParser := getItemTypeParser(itemsSchema)

	for {
		end := skipCSVDelimitersOrWhiteSpaces(rawValue)
		if end >= len(rawValue) {
			return
		}
		rawValue = rawValue[end:]

		var item any
		item, end, err = itemParser(rawValue)
		if err != nil {
			return
		}
		rawValue = rawValue[end:]

		value = append(value, item)
	}
}

func parseArrayFlagValueSingle(itemsSchema *core.Schema, rawValue string) (value []any, err error) {
	value, err = parseJSONFlagValue[[]any](rawValue)
	if err == nil {
		return
	}

	value, csvErr := parseArrayCSV(itemsSchema, rawValue)
	if csvErr == nil {
		err = nil
		return
	}

	return
}

func parseArrayFlagValue(itemsSchema *core.Schema, rawValues []string) (items []any, err error) {
	for i, rawValue := range rawValues {
		value, err := parseArrayFlagValueSingle(itemsSchema, rawValue)
		if err != nil {
			if len(rawValues) > 0 {
				err = &utils.ChainedError{Name: fmt.Sprint(i), Err: err}
			}
			return items, err
		}
		items = append(items, value...)
	}

	return
}

func parseObjectKey(s string) (key string, end int, err error) {
	return parseString(s, isObjectKeyDelimiter)
}

func parseObjectValue(propSchema *core.Schema, s string) (value any, end int, err error) {
	start := skipWhiteSpaces(s)
	if start >= len(s) {
		return
	}

	itemParser := getItemTypeParser(propSchema)
	value, end, err = itemParser(s[start:])
	end += start
	if err != nil {
		return
	}

	return
}

func getSchemaFromAlternatives(refs ...*mgcSchemaPkg.SchemaRef) *core.Schema {
	for _, ref := range refs {
		if ref != nil && ref.Value != nil {
			return (*mgcSchemaPkg.Schema)(ref.Value)
		}
	}

	return nil
}

func parseObjectCSV(schema *core.Schema, rawValue string) (value map[string]any, err error) {
	for {
		end := skipCSVDelimitersOrWhiteSpaces(rawValue)
		if end >= len(rawValue) {
			return
		}
		rawValue = rawValue[end:]

		var propName string
		propName, end, err = parseObjectKey(rawValue)
		if err != nil {
			return
		}
		rawValue = rawValue[end:]

		propSchema := getSchemaFromAlternatives(
			schema.Properties[propName],
			schema.AdditionalProperties.Schema,
		)

		var propValue any
		propValue, end, err = parseObjectValue(propSchema, rawValue)
		if err != nil {
			return
		}
		rawValue = rawValue[end:]

		if value == nil {
			value = map[string]any{}
		}
		value[propName] = propValue
	}
}

func parseObjectFlagValueSingle(schema *core.Schema, rawValue string) (value map[string]any, err error) {
	value, err = parseJSONFlagValue[map[string]any](rawValue)
	if err == nil {
		return
	}

	value, csvErr := parseObjectCSV(schema, rawValue)
	if csvErr == nil {
		err = nil
		return
	}

	return
}

func parseObjectFlagValue(schema *core.Schema, rawValues []string) (items map[string]any, err error) {
	for i, rawValue := range rawValues {
		value, err := parseObjectFlagValueSingle(schema, rawValue)
		if err != nil {
			if len(rawValues) > 0 {
				err = &utils.ChainedError{Name: fmt.Sprint(i), Err: err}
			}
			return items, err
		}
		if len(value) == 0 {
			continue
		}

		if items == nil {
			items = map[string]any{}
		}

		maps.Copy(items, value)
	}

	return
}

func loadJSONFromFile[T any](filename string) (value T, err error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}

	err = json.Unmarshal(data, &value)
	return
}

func loadVerbatimFromFile(filename string) (value string, err error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}

	value = string(data)
	return
}
