package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"
)

type dummyTransport struct{}

func (o dummyTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, nil
}

func TestClientCreation(t *testing.T) {
	client := NewClient(dummyTransport{})
	if client == nil {
		t.Fail()
	}
}

func TestContext(t *testing.T) {
	ctx := context.Background()
	if ClientFromContext(ctx) != nil {
		t.Error("corehttp.ClientFromContext() should not return a client from an empty context")
	}
	client := NewClient(dummyTransport{})
	ctx = NewClientContext(ctx, client)
	if ClientFromContext(ctx) == nil {
		t.Error("corehttp.ClientFromContext() failed to retrieve client from valid context")
	}
}

type dummyResponseBodyStruct struct {
	Data string `json:"data"`
}

func TestDecodeJSON(t *testing.T) {
	expectedData := "some string"
	dummyResponse := &http.Response{
		Body: io.NopCloser(bytes.NewBufferString(fmt.Sprintf("{\"data\": \"%s\"}", expectedData))),
	}
	decoded := new(dummyResponseBodyStruct)
	err := DecodeJSON(dummyResponse, &decoded)
	if err != nil {
		t.Errorf("DecodeJSON function failed: %s", err)
	}
	if decoded.Data != "some string" {
		t.Errorf("DecodeJSON function failed. 'dummyResponseBodyStruct.Data' expected %s but got %s", expectedData, decoded.Data)
	}
}

func TestNewHttpErrorFromResponse(t *testing.T) {
	dummyResponse := &http.Response{
		Body:       io.NopCloser(bytes.NewBufferString("some value")),
		StatusCode: 123,
		Status:     "not ok",
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
	err := NewHttpErrorFromResponse(dummyResponse)
	httpErr, ok := err.(*HttpError)
	if !ok {
		t.Error("NewHttpErrorFromResponse did not return expected HttpError type")
	}

	expected := &HttpError{
		Code:    123,
		Status:  "not ok",
		Headers: http.Header{"Content-Type": []string{"application/json"}},
		Payload: bytes.NewBufferString("some value").Bytes(),
		Message: "not ok",
		Slug:    "unknown",
	}
	if !reflect.DeepEqual(httpErr, expected) {
		t.Errorf("NewHttpErrorFromResponse returned %+v, but expected %+v", *httpErr, *expected)
	}

	dummyResponse.Body = io.NopCloser(bytes.NewBufferString("{\"slug\": \"the slug\",\"message\": \"the message\"}"))
	expected.Message = "the message"
	expected.Slug = "the slug"
	expected.Payload = bytes.NewBufferString("{\"slug\": \"the slug\",\"message\": \"the message\"}").Bytes()

	httpErr = NewHttpErrorFromResponse(dummyResponse).(*HttpError)
	if !reflect.DeepEqual(httpErr, expected) {
		t.Errorf("NewHttpErrorFromResponse failed to decode response's 'data' and 'message' fields properly\nInput: %+v\nOutput: %+v\nExpected: %+v", *dummyResponse, *httpErr, *expected)
	}
}
