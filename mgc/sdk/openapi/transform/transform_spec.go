package transform

import (
	"github.com/getkin/kin-openapi/openapi3"
	"magalu.cloud/core/utils"
)

type transformSpec struct {
	Type string `json:"type" yaml:"type"`
	// See more about the 'remain' directive here: https://pkg.go.dev/github.com/mitchellh/mapstructure#hdr-Remainder_Values
	Parameters map[string]any `json:",remain"` // nolint
	Schema     *openapi3.Schema
}

func newTransformSpecFromString(s string) *transformSpec {
	if len(s) == 0 {
		return nil
	}
	return &transformSpec{Type: s}
}

func newTransformSpecFromMap(m map[string]any) *transformSpec {
	if len(m) == 0 {
		return nil
	}
	spec, err := utils.DecodeNewValue[transformSpec](m)
	if err != nil || len(spec.Type) == 0 {
		return nil
	}
	return spec
}

// spec must be string or map
func newTransformSpec(spec any) *transformSpec {
	if s, ok := spec.(string); ok {
		return newTransformSpecFromString(s)
	} else if m, ok := spec.(map[string]any); ok {
		return newTransformSpecFromMap(m)
	}
	return nil
}

func newTransformSpecSlice(sl []any) []*transformSpec {
	ret := make([]*transformSpec, 0, len(sl))
	for _, spec := range sl {
		if ts := newTransformSpec(spec); ts != nil {
			ret = append(ret, ts)
		}
	}
	if len(ret) == 0 {
		return nil
	}
	return ret
}

func getTransformationSpecs(extensions map[string]any, transformationKey string) []*transformSpec {
	if spec, ok := extensions[transformationKey]; !ok {
		return nil
	} else if sl, ok := spec.([]any); ok {
		return newTransformSpecSlice(sl)
	} else if ts := newTransformSpec(spec); ts != nil {
		return []*transformSpec{ts}
	} else {
		return nil
	}
}
