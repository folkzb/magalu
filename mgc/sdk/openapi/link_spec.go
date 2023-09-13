package openapi

import (
	"magalu.cloud/core"
	corehttp "magalu.cloud/core/http"
)

type linkSpecResolver struct {
	httpResult         corehttp.HttpResult
	findParameterValue func(location, name string) (core.Value, bool)
}

func (s *linkSpecResolver) resolve(value core.Value) (core.Value, bool, error) {
	switch specVal := value.(type) {
	case string:
		rtExp := linkRtExpression{specVal, s.httpResult, s.findParameterValue}
		return rtExp.resolve()
	default:
		// Treat as raw value
		return specVal, true, nil
	}
}
