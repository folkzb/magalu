package core

import (
	"errors"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

func logger() *zap.SugaredLogger {
	if pkgLogger == nil {
		pkgLogger = initPkgLogger().Named("http")
	}
	return pkgLogger
}

type HttpClientLogger struct {
	Transport http.RoundTripper
}

func NewDefaultHttpClientLogger(transport http.RoundTripper) *HttpClientLogger {
	return &HttpClientLogger{
		Transport: transport,
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
	log.Debugw("request header info", "headers", LogHttpHeaders(req.Header))
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

	log.Debugw("response header info", "headers", LogHttpHeaders(resp.Header))
}
