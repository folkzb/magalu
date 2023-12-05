package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"go.uber.org/zap"
)

type ClientLogger struct {
	Transport http.RoundTripper
}

func NewDefaultClientLogger(transport http.RoundTripper) *ClientLogger {
	return &ClientLogger{
		Transport: transport,
	}
}

func (t *ClientLogger) RoundTrip(req *http.Request) (*http.Response, error) {
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

func (t *ClientLogger) logRequest(log *zap.SugaredLogger, req *http.Request) {
	log.Debugw("request", "headers", LogHttpHeaders(req.Header))
}

func (t *ClientLogger) logResponse(log *zap.SugaredLogger, req *http.Request, resp *http.Response, err error) {
	if resp == nil {
		if err == nil {
			err = errors.New("Unknown Error")
		}
		log.Debugw("request error", "error", err)
		return
	}
	log = log.With("headers", LogHttpHeaders(resp.Header))
	if err != nil {
		log.Debugw("response with error", "status", resp.Status, "error", err)
	} else {
		log.Debugw("response", "status", resp.Status)
	}
}

type LogRequest http.Request

func (r LogRequest) MarshalJSON() ([]byte, error) {
	var url any
	if r.URL != nil {
		url = r.URL.String()
	}
	return json.Marshal(map[string]any{
		"method":   r.Method,
		"url":      url,
		"protocol": r.Proto,
		"headers":  LogHttpHeaders(r.Header),
	})
}

type LogResponse http.Response

func (r LogResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"status":   r.Status,
		"protocol": r.Proto,
		"headers":  LogHttpHeaders(r.Header),
	})
}
