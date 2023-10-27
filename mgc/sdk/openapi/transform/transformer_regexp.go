package transform

import (
	"fmt"
	"regexp"

	"magalu.cloud/core/utils"
)

type transformRegExpSpec struct {
	// Regular Expression as per https://pkg.go.dev/regexp#Compile
	Pattern string `json:"pattern" yaml:"pattern"`
	// Replacement Template as per https://pkg.go.dev/regexp#Regexp.Expand
	Replacement string `json:"replacement" yaml:"replacement"`
}

func transformRegExp(params map[string]any, s string) (result string, err error) {
	spec, err := utils.DecodeNewValue[transformRegExpSpec](params)
	if err != nil {
		return s, fmt.Errorf("invalid regexp parameters: %w", err)
	}
	if len(spec.Pattern) == 0 {
		return s, fmt.Errorf("invalid regexp parameters: missing pattern")
	}
	re, err := regexp.Compile(spec.Pattern)
	if err != nil {
		return s, fmt.Errorf("invalid regexp pattern %q: %w", spec.Pattern, err)
	}

	b := []byte{}
	for _, submatches := range re.FindAllStringSubmatchIndex(s, -1) {
		b = re.ExpandString(b, spec.Replacement, s, submatches)
	}
	return string(b), nil
}
