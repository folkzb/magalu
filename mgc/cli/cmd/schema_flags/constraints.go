package schema_flags

import (
	"encoding/json"
	"fmt"
	"slices"
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
	case "integer", "number":
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

	if s.Pattern != "" {
		*dst = append(*dst, fmt.Sprintf("pattern: %s", s.Pattern))
	}
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
}

func addArrayConstraints(s *mgcSdk.Schema, dst *[]string) {
	if s.MinItems != 0 && s.MaxItems != nil {
		*dst = append(*dst, fmt.Sprintf("between %v and %v items", s.MinItems, int(*s.MaxItems)))
	} else if s.MinItems != 0 {
		*dst = append(*dst, fmt.Sprintf("at least %v items", s.MinItems))
	} else if s.MaxItems != nil {
		*dst = append(*dst, fmt.Sprintf("at most %v items", int(*s.MaxItems)))
	}
}

func addObjectConstraints(s *mgcSdk.Schema, dst *[]string) {
	if s.AdditionalProperties.Has != nil && *s.AdditionalProperties.Has {
		return
	}

	if len(s.Properties) == 0 {
		return
	}

	keys := []string{}
	for k := range s.Properties {
		keys = append(keys, k)
	}

	slices.Sort(keys)
	*dst = append(*dst, formatAlternatives("single property: %s", "properties: %s and %s", keys))
}

func getEnumAsString(s *mgcSdk.Schema) (asStrings []string) {
	length := len(s.Enum)
	if length == 0 {
		return
	}

	asStrings = make([]string, 0, length)
	for _, e := range s.Enum {
		data, err := json.Marshal(e)
		var s string
		if err != nil {
			s = fmt.Sprint(e)
		} else {
			s = string(data)
		}

		asStrings = append(asStrings, s)
	}

	slices.Sort(asStrings)

	return
}

func addEnumConstraint(s *mgcSdk.Schema, dst *[]string) {
	asStrings := getEnumAsString(s)
	if len(asStrings) == 0 {
		return
	}

	*dst = append(*dst, formatAlternatives("must be %s", "one of %s or %s", asStrings))
}

// oneFmt takes a single parameter, while multipleFmt takes exactly 2
func formatAlternatives(oneFmt string, multipleFmt string, asStrings []string) string {
	switch len(asStrings) {
	case 0:
		return ""

	case 1:
		return fmt.Sprintf(oneFmt, asStrings[0])

	default:
		lastIndex := len(asStrings) - 1
		commaDelimited := strings.Join(asStrings[:lastIndex], ", ")
		return fmt.Sprintf(multipleFmt, commaDelimited, asStrings[lastIndex])
	}
}

func getDescriptionConstraints(s *mgcSdk.Schema) string {
	constraints := []string{}

	if filler, ok := getSchemaConstraintFiller(s.Type); ok {
		filler(s, &constraints)
	}

	addEnumConstraint(s, &constraints)
	return formatAlternatives("%s", "%s and %s", constraints)
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

func shouldRecommendHelpValue(s *mgcSdk.Schema) bool {
	switch s.Type {
	case "integer", "number", "boolean", "string":
		return false

	case "array":
		if s.Items != nil && s.Items.Value != nil {
			return shouldRecommendHelpValue((*mgcSdk.Schema)(s.Items.Value))
		}
		return true

	case "object":
		return true

	default:
		return true
	}
}
