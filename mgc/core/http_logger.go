package core

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

func logger() *zap.SugaredLogger {
	if pkgLogger == nil {
		pkgLogger = initPkgLogger().Named("http")
	}
	return pkgLogger
}

type HttpClientLogger struct {
	Transport    http.RoundTripper
	RedactHeader func(req *http.Request, key string, value string) string
}

func NewDefaultHttpClientLogger(transport http.RoundTripper) *HttpClientLogger {
	return &HttpClientLogger{
		Transport:    transport,
		RedactHeader: DefaultHttpClientRedactHeader,
	}
}

func (t *HttpClientLogger) RoundTrip(req *http.Request) (*http.Response, error) {
	log := logger().With("method", req.Method, "url", req.URL, "protocol", req.Proto)
	t.logRequest(log, req)

	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	resp, err := transport.RoundTrip(req)

	t.logResponse(log, req, resp, err)

	return resp, err
}

func (t *HttpClientLogger) logRequest(log *zap.SugaredLogger, req *http.Request) {
	log.Debugw("will send HTTP request")
	for k, lst := range req.Header {
		for _, v := range lst {
			if t.RedactHeader != nil {
				v = t.RedactHeader(req, k, v)
			}
			// TODO: log all header information using Infow(..., "headers", LogHttpHeaders(req.Header))
			logger().Debugw("request header info", "key", k, "value", v)
		}
	}
}

func (t *HttpClientLogger) logResponse(log *zap.SugaredLogger, req *http.Request, resp *http.Response, err error) {
	if resp == nil {
		if err == nil {
			err = errors.New("Unknown Error")
		}
		log.Debugw("requested ended with error", "error", err.Error())
		return
	}
	errMsg := ""
	if err != nil {
		errMsg = fmt.Sprintf("; error: %s", err.Error())
		log.Debugw("received HTTP response with error", "status", resp.Status, "error", errMsg)
	} else {
		log.Debugw("received HTTP response", "status", resp.Status)
	}

	for k, lst := range resp.Header {
		for _, v := range lst {
			if t.RedactHeader != nil {
				v = t.RedactHeader(req, k, v)
			}
			// TODO: log all header information using Infow(..., "headers", LogHttpHeaders(resp.Header))
			logger().Debugw("response header info", "key", k, "value", v)
		}
	}
}

func DefaultHttpClientRedactHeader(req *http.Request, key string, value string) string {
	if strings.ToLower(key) == "authorization" {
		parts := strings.SplitN(value, " ", 2)
		if len(parts) > 1 {
			if strings.ToLower(parts[0]) == "bearer" {
				parts[1] = fmt.Sprintf("<redacted %d chars>", len(parts[1]))
			}
		}
		return strings.Join(parts, " ")
	}
	return value
}
