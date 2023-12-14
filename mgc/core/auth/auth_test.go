package auth

import (
	"fmt"
	"io"
	"net/http"
)

var dummyConfig Config = Config{
	ClientId:       "client-id",
	RedirectUri:    "redirect-uri",
	LoginUrl:       "login-url",
	TokenUrl:       "token-url",
	ValidationUrl:  "validation-url",
	RefreshUrl:     "refresh-url",
	TenantsListUrl: "tenant-list-url",
	Scopes:         []string{"test"},
}

var dummyConfigResult *ConfigResult = &ConfigResult{
	AccessToken:     "access-token",
	RefreshToken:    "refresh-token",
	CurrentTenantID: "tenant-id",
	CurrentEnv:      "test",
}

var dummyConfigResultYaml = []byte(`---
access_token: "access-token"
refresh_token: "refresh-token"
current_tenant_id: "tenant-id"
current_environment: "test"
`)

type mockTransport struct {
	statusCode        int
	responseBody      io.ReadCloser
	shouldReturnError bool
}

func (o mockTransport) RoundTrip(*http.Request) (*http.Response, error) {
	if o.shouldReturnError {
		return nil, fmt.Errorf("test error")
	}
	return &http.Response{StatusCode: o.statusCode, Body: o.responseBody}, nil
}
