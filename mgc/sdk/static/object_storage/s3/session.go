package s3

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"magalu.cloud/core"
	corehttp "magalu.cloud/core/http"
)

var excludedHeaders = HeaderMap{
	"Authorization":         nil,
	"Accept-Encoding":       nil,
	"Amz-Sdk-Invocation-Id": nil,
	"Amz-Sdk-Request":       nil,
	"User-Agent":            nil,
	"X-Amzn-Trace-Id":       nil,
	"Expect":                nil,
}

func BuildHost(cfg Config) string {
	if cfg.ServerUrl != "" {
		return cfg.ServerUrl
	}
	return strings.ReplaceAll(templateUrl, "{{region}}", cfg.Region)
}

func SendRequest[T core.Value](ctx context.Context, req *http.Request, accessKey, secretKey string) (result T, err error) {
	httpClient := corehttp.ClientFromContext(ctx)
	if httpClient == nil {
		err = fmt.Errorf("couldn't get http client from context")
		return
	}

	var unsignedPayload bool
	switch req.Body.(type) {
	case io.ReadSeeker:
		unsignedPayload = true
	}

	if err = sign(req, accessKey, secretKey, unsignedPayload, excludedHeaders); err != nil {
		return
	}

	res, err := httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("error to send HTTP request: %w", err)
		return
	}

	result, err = corehttp.UnwrapResponse[T](res)
	return
}
