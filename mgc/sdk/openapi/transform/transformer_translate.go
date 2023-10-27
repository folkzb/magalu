package transform

import (
	"fmt"
	"reflect"

	"magalu.cloud/core/utils"
)

type transformTranslateSpecItem struct {
	From any `json:"from"`
	To   any `json:"to"`
}

type transformTranslateSpec struct {
	Translations []transformTranslateSpecItem `json:"translations"`
	AllowMissing bool                         `json:"allowMissing,omitempty"`
}

func transformTranslate(params map[string]any, value any) (result any, err error) {
	spec, err := utils.DecodeNewValue[transformTranslateSpec](params)
	if err != nil {
		return value, fmt.Errorf("invalid translation parameters: %w", err)
	}
	if len(spec.Translations) == 0 {
		return value, fmt.Errorf("invalid translation parameters: missing translations")
	}
	for _, item := range spec.Translations {
		if reflect.DeepEqual(item.From, value) {
			return item.To, nil
		}
	}
	if spec.AllowMissing {
		return value, nil
	}
	return value, fmt.Errorf("translation not found: %+v", value)
}
