package cmd

import (
	"fmt"
	"strconv"
	"strings"

	mgcSdk "magalu.cloud/sdk"
)

type schemaConstraintFiller func(s *mgcSdk.Schema, dst *[]string)

// Don't use in-memory map because of initialization cycle due to array filler recursion
func getSchemaConstraintFiller(t string) (schemaConstraintFiller, bool) {
	switch t {
	case "string":
		return addStringConstraints, true
	case "integer":
		return addNumberConstraints, true
	case "array":
		return addArrayConstraints, true
	case "object":
		return addObjectConstraints, true
	}

	return nil, false
}

func addStringConstraints(s *mgcSdk.Schema, dst *[]string) {
	if s.MinLength != 0 && s.MaxLength != nil {
		*dst = append(*dst, fmt.Sprintf("between %v and %v characters", s.MinLength, int(*s.MaxLength)))
	} else if s.MinLength != 0 {
		*dst = append(*dst, fmt.Sprintf("min character count: %v", s.MinLength))
	} else if s.MaxLength != nil {
		*dst = append(*dst, fmt.Sprintf("max character count: %v", int(*s.MaxLength)))
	}

	if s.Format != "" {
		*dst = append(*dst, "format: "+s.Format)
	}

	addEnumConstraint(s, dst)
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func addNumberConstraints(s *mgcSdk.Schema, dst *[]string) {
	min := getMinNumber(s)
	max := getMaxNumber(s)

	if min != nil && max != nil {
		*dst = append(*dst, fmt.Sprintf("range: %s - %s", formatFloat(*min), formatFloat(*max)))
	} else if min != nil {
		*dst = append(*dst, fmt.Sprintf("min: %s", formatFloat(*min)))
	} else if max != nil {
		*dst = append(*dst, fmt.Sprintf("max: %s", formatFloat(*max)))
	}

	addEnumConstraint(s, dst)
}

func addArrayConstraints(s *mgcSdk.Schema, dst *[]string) {
	if s.MinItems != 0 && s.MaxItems != nil {
		*dst = append(*dst, fmt.Sprintf("between %v and %v items", s.MinItems, int(*s.MaxItems)))
	} else if s.MinItems != 0 {
		*dst = append(*dst, fmt.Sprintf("at least %v items", s.MinItems))
	} else if s.MaxItems != nil {
		*dst = append(*dst, fmt.Sprintf("at most %v items", int(*s.MaxItems)))
	}

	if s.Items != nil {
		items := (*mgcSdk.Schema)(s.Items.Value)
		if filler, ok := getSchemaConstraintFiller(items.Type); ok {
			itemsConstraints := []string{}
			filler(items, &itemsConstraints)
			if len(itemsConstraints) > 0 {
				itemsStr := fmt.Sprintf("%s:(%s)", items.Type, strings.Join(itemsConstraints, " and "))
				*dst = append(*dst, itemsStr)
			}
		}
	}
}

func getSchemaJSONrepresentation(s *mgcSdk.Schema) string {
	switch t := s.Type; t {
	case "array":
		if s.Items != nil {
			return fmt.Sprintf("array(%s)", getSchemaJSONrepresentation((*mgcSdk.Schema)(s.Items.Value)))
		} else {
			return "array"
		}
	case "object":
		result := "{"
		i := 0

		for name, prop := range s.Properties {
			prefix := " "
			if i > 0 {
				prefix = ", "
			}

			result += fmt.Sprintf("%s%s: %s", prefix, name, getSchemaJSONrepresentation((*mgcSdk.Schema)(prop.Value)))
			i++
		}
		result += " }"
		return result
	default:
		return t
	}
}

func addObjectConstraints(s *mgcSdk.Schema, dst *[]string) {
	propLen := len(s.Properties)
	if propLen == 0 {
		return
	}

	*dst = append(*dst, getSchemaJSONrepresentation(s))
}

func addEnumConstraint(s *mgcSdk.Schema, dst *[]string) {
	length := len(s.Enum)
	if length == 0 {
		return
	}

	asStrings := make([]string, 0, length)
	for _, e := range s.Enum {
		asStrings = append(asStrings, fmt.Sprintf("%v", e))
	}

	*dst = append(*dst, fmt.Sprintf("one of %v", strings.Join(asStrings, "|")))
}

func schemaValueConstraints(s *mgcSdk.Schema) string {
	constraints := []string{}

	if filler, ok := getSchemaConstraintFiller(s.Type); ok {
		filler(s, &constraints)
	}

	return strings.Join(constraints, " and ")
}

func getMinNumber(s *mgcSdk.Schema) *float64 {
	if s.Min == nil {
		return nil
	}

	minInt := *s.Min
	if s.ExclusiveMin {
		minInt += 1
	}

	return &minInt
}

func getMaxNumber(s *mgcSdk.Schema) *float64 {
	if s.Max == nil {
		return nil
	}

	maxInt := *s.Max
	if s.ExclusiveMax {
		maxInt += 1
	}

	return &maxInt
}
