package sdk

import (
	"net/http"

	"github.com/google/uuid"
)

var _ http.RoundTripper = (*DefaultSdkTransport)(nil)

type DefaultSdkTransport struct {
	Transport http.RoundTripper
	UserAgent string
}

func newDefaultSdkTransport(transport http.RoundTripper, userAgent string) *DefaultSdkTransport {
	return &DefaultSdkTransport{
		Transport: transport,
		UserAgent: userAgent,
	}
}

func (t *DefaultSdkTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", t.UserAgent)
	req.Header.Set("X-Request-Id", uuid.New().String())

	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	resp, err := transport.RoundTrip(req)

	return resp, err
}
