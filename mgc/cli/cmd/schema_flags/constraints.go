package schema_flags

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
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

func getXOfJSONRepresentation(xOf openapi3.SchemaRefs, xOfType string) string {
	xOfs := make([]string, len(xOf))
	for i, sub := range xOf {
		xOfs[i] = getSchemaJSONrepresentation((*mgcSdk.Schema)(sub.Value))
	}
	return fmt.Sprintf("%s(%s)", xOfType, strings.Join(xOfs, ", "))
}

func getArrayJSONRepresentation(s *mgcSdk.Schema) string {
	if s.Items != nil {
		return fmt.Sprintf("[%s]", schemaJSONRepAndConstraints((*mgcSdk.Schema)(s.Items.Value), true))
	} else {
		return "[]"
	}
}

func getXOfJSONRepresentations(s *mgcSdk.Schema) string {
	allReps := []string{}
	if len(s.AnyOf) > 0 {
		allReps = append(allReps, getXOfJSONRepresentation(s.AnyOf, "anyOf"))
	}
	if len(s.AllOf) > 0 {
		allReps = append(allReps, getXOfJSONRepresentation(s.AllOf, "allOf"))
	}
	if len(s.OneOf) > 0 {
		allReps = append(allReps, getXOfJSONRepresentation(s.OneOf, "oneOf"))
	}

	return strings.Join(allReps, ", ")
}

func getObjectJSONRepresentation(s *mgcSdk.Schema) string {
	propReps := make([]string, 0, len(s.Properties))

	for name, prop := range s.Properties {
		propRep := fmt.Sprintf("%q: %s", name, schemaJSONRepAndConstraints((*mgcSdk.Schema)(prop.Value), true))
		if prop.Value.Default != nil {
			propRep += fmt.Sprintf(" (default %v)", prop.Value.Default)
		}

		propReps = append(propReps, propRep)
	}

	slices.Sort(propReps)

	return "{" + strings.Join(propReps, ", ") + "}"
}

func getSchemaJSONrepresentation(s *mgcSdk.Schema) string {
	allReps := []string{}

	if xOfReps := getXOfJSONRepresentations(s); len(xOfReps) > 1 {
		allReps = append(allReps, xOfReps)
	}

	switch t := s.Type; t {
	case "array":
		allReps = append(allReps, getArrayJSONRepresentation(s))
	case "object":
		if len(s.Properties) > 0 {
			allReps = append(allReps, getObjectJSONRepresentation(s))
		}
	case "":
		break
	default:
		allReps = append(allReps, t)
	}

	return strings.Join(allReps, ", ")
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

	slices.Sort(asStrings)

	*dst = append(*dst, fmt.Sprintf("one of %v", strings.Join(asStrings, "|")))
}

func isJSONRepresentationNeeded(s *mgcSdk.Schema, representation string) bool {
	if representation == s.Type {
		return false
	}

	if s.Items != nil {
		return representation != fmt.Sprintf("[%s]", s.Items.Value.Type) && representation != "[]"
	}

	return true
}

// If the JSON representation is simple (for simple types like 'string' or for simple arrays like '[string]'),
// only the constraints are returned. If the JSON representation is more complex (for objects, object arrays, xOfs...),
// both the JSON representation and constraints are returned.
// To always return the JSON representation regardless, set 'includeSimpleReturn' to true
func schemaJSONRepAndConstraints(s *mgcSdk.Schema, includeSimpleReturn bool) string {
	constraints := []string{}

	if filler, ok := getSchemaConstraintFiller(s.Type); ok {
		filler(s, &constraints)
	}

	addEnumConstraint(s, &constraints)

	if jsonRepresentation := getSchemaJSONrepresentation(s); includeSimpleReturn || isJSONRepresentationNeeded(s, jsonRepresentation) {
		if len(constraints) > 0 {
			return fmt.Sprintf("%s (%s)", jsonRepresentation, strings.Join(constraints, " and "))
		}
		return jsonRepresentation
	} else {
		return strings.Join(constraints, " and ")
	}
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
