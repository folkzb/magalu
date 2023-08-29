package http

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDefaultBackoff(t *testing.T) {
	const retryAfter = time.Duration(10000000000)
	dummyResponse := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Retry-After": []string{fmt.Sprint(retryAfter)}},
	}
	min := time.Duration(1000000000)
	max := time.Duration(10000000000)
	attemptNum := 5
	duration := DefaultBackoff(min, max, attemptNum, dummyResponse)
	if duration != retryAfter {
		t.Errorf("DefaultBackoff ignored value in response even though status was 'too many requests', expected 10000000000 but got %v", duration)
	}

	dummyResponse.StatusCode = http.StatusServiceUnavailable
	duration = DefaultBackoff(min, max, attemptNum, dummyResponse)
	if duration != retryAfter {
		t.Errorf("DefaultBackoff ignored value in response even though status was 'service unavailable', expected 10000000000 but got %v", duration)
	}

	dummyResponse.Header = http.Header{}
	duration = DefaultBackoff(min, max, attemptNum, dummyResponse)
	if duration != max {
		t.Errorf("DefaultBackoff calculations failed. min: %v, max: %v, attemptNum: %v. Expected %v but got %v", min, max, attemptNum, max, duration)
	}
	min = time.Duration(10)
	duration = DefaultBackoff(min, max, attemptNum, dummyResponse)
	expected := time.Duration(math.Pow(2, float64(attemptNum)) * float64(min))
	if duration != expected {
		t.Errorf("DefaultBackoff calculations failed. min: %v, max: %v, attemptNum: %v. Expected %v but got %v", min, max, attemptNum, expected, duration)
	}
}

type refreshTransportTestCase struct {
	returnUnauthorized bool
	returnError        bool
}

func (t *refreshTransportTestCase) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.returnError {
		return nil, fmt.Errorf("some error")
	}

	if t.returnUnauthorized {
		return &http.Response{StatusCode: http.StatusUnauthorized}, nil
	} else {
		return &http.Response{StatusCode: http.StatusOK}, nil
	}
}

func TestRoundTrip(t *testing.T) {
	shouldFailRefreshFn := new(bool)
	refreshCallCount := new(int)
	*shouldFailRefreshFn = true
	*refreshCallCount = 0

	transport := &refreshTransportTestCase{returnUnauthorized: false, returnError: false}
	refreshFn := func(context.Context) (string, error) {
		*refreshCallCount += 1
		if *shouldFailRefreshFn {
			return "", fmt.Errorf("expected fail")
		} else {
			transport.returnError = false
			transport.returnUnauthorized = false
			return "valid token", nil
		}
	}

	logger := NewDefaultRefreshLogger(transport, refreshFn)
	baseReq := &http.Request{}
	resp, _ := logger.RoundTrip(baseReq)
	expectedResp := &http.Response{StatusCode: http.StatusOK}
	if !reflect.DeepEqual(resp, expectedResp) {
		t.Error("RefreshLogger.RoundTrip didn't passthrough response when authorization is unset")
	}
	if *refreshCallCount != 0 {
		t.Error("RefreshLogger.RoundTrip called RefreshFn when authorization is unset")
	}

	baseReq = &http.Request{Header: http.Header{"Authorization": []string{"valid token"}}}
	resp, _ = logger.RoundTrip(baseReq)
	if !reflect.DeepEqual(resp, expectedResp) {
		t.Error("RefreshLogger.RoundTrip didn't passthrough response when status is ok")
	}
	if *refreshCallCount != 0 {
		t.Error("RefreshLogger.RoundTrip called RefreshFn when status is ok")
	}

	transport.returnError = true
	expectedResp = nil
	resp, _ = logger.RoundTrip(baseReq)
	if !reflect.DeepEqual(resp, expectedResp) {
		t.Error("RefreshLogger.RoundTrip didn't passthrough response when transport returned error")
	}
	if *refreshCallCount != 0 {
		t.Error("RefreshLogger.RoundTrip called RefreshFn when transport returned error")
	}

	transport.returnError = false
	transport.returnUnauthorized = true
	*shouldFailRefreshFn = true
	baseReq = &http.Request{Header: http.Header{"Authorization": []string{"expired token"}}}
	_, err := logger.RoundTrip(baseReq)
	if *refreshCallCount != 1 {
		t.Error("RefreshLogger.RoundTrip didn't try to refresh when expected")
	}
	if err == nil {
		t.Error("RefreshLogger.RoundTrip didn't return error, even though refresh failed")
	}
	if baseReq.Header.Get("Authorization") != "expired token" {
		t.Error("RefreshLogger.RoundTrip modified request authorization header, even though refresh failed")
	}

	*shouldFailRefreshFn = false
	baseReq = &http.Request{Header: http.Header{"Authorization": []string{"expired token"}}}
	resp, err = logger.RoundTrip(baseReq)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Error("RefreshLogger.RoundTrip returned unexpected error")
	}
	if *refreshCallCount != 2 {
		t.Error("RefreshLogger.RoundTrip didn't try to refresh when expected")
	}
	if !strings.HasPrefix(baseReq.Header.Get("Authorization"), "Bearer") {
		t.Error("RefreshLogger.RoundTrip didn't re-set authorization header after refresh")
	}
}
