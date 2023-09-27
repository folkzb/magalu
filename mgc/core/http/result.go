package http

import (
	"io"
	"mime/multipart"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"magalu.cloud/core"
)

type HttpResult interface {
	core.Result
	Request() *http.Request
	RequestBody() any // pre-Marshal, the original structured body data, if any. If request used an io.Reader, this is nil
	Response() *http.Response
	ResponseBody() any // post-Unmarshal, the decoded structured body data, if any; or io.Reader if not a structured body data
}

type httpResult struct {
	source       core.ResultSource
	request      *http.Request
	requestBody  any // pre-Marshal, the original structured body data, if any. If request used an io.Reader, this is nil
	response     *http.Response
	responseBody any // post-Unmarshal, the decoded structured body data, if any; or io.Reader if not a structured body data
}

type httpResultWithValue struct {
	httpResult
	schema *core.Schema
	value  core.Value
}

type httpResultWithReader struct {
	httpResult
	reader io.Reader
}

type httpResultWithMultipart struct {
	httpResult
	multipart *multipart.Part
}

// Takes over a response, unwrap it and create a Result based on it.
//
// The requestBody is the structured body data before it was marshalled to bytes. If the
// request was built from an io.Reader, such as file uploads or multipart, then this should be nil.
//
// The result is heavily dependent on the output of UnwrapResponse(), if data is:
//   - io.Reader: then implements ResultWithReader and Reader() returns it;
//   - multipart.Part: then implements ResultWithMultipart and Multipart() returns it;
//   - else: implements ResultWithValue and the decoded structured data is returned by Value().
//     It may be transformed/converted with getValueFromResponseBody() if it's non-nil.
func NewHttpResult(
	source core.ResultSource,
	schema *core.Schema,
	request *http.Request,
	requestBody any, // pre-Marshal, the original JSON, if any
	response *http.Response,
	getValueFromResponseBody func(responseBody any) (core.Value, error),
) (r HttpResult, err error) {
	result := httpResult{
		source:      source,
		request:     request,
		response:    response,
		requestBody: requestBody,
	}

	result.responseBody, err = UnwrapResponse[any](response)
	if err != nil {
		return
	}

	switch v := result.responseBody.(type) {
	case *multipart.Part:
		return &httpResultWithMultipart{result, v}, nil
	case io.Reader:
		return &httpResultWithReader{result, v}, nil
	default:
		var value core.Value
		if getValueFromResponseBody == nil {
			value = v
		} else {
			value, err = getValueFromResponseBody(v)
			if err != nil {
				return
			}
		}
		return &httpResultWithValue{result, schema, value}, nil
	}
}

func (r *httpResult) Source() core.ResultSource {
	return r.source
}

func (r *httpResult) Request() *http.Request {
	return r.request
}

func (r *httpResult) RequestBody() any {
	return r.requestBody
}

func (r *httpResult) Response() *http.Response {
	return r.response
}

func (r *httpResult) ResponseBody() any {
	return r.responseBody
}

var _ HttpResult = (*httpResult)(nil)

func (r *httpResultWithValue) Unwrap() core.Result {
	return &r.httpResult
}

func (r *httpResultWithValue) Schema() *core.Schema {
	return r.schema
}

func (r *httpResultWithValue) ValidateSchema() error {
	return r.schema.VisitJSON(r.value, openapi3.MultiErrors())
}

func (r *httpResultWithValue) Value() core.Value {
	return r.value
}

var _ HttpResult = (*httpResultWithValue)(nil)
var _ core.ResultWrapper = (*httpResultWithValue)(nil)
var _ core.ResultWithValue = (*httpResultWithValue)(nil)

func (r *httpResultWithReader) Unwrap() core.Result {
	return &r.httpResult
}

func (r *httpResultWithReader) Reader() io.Reader {
	return r.reader
}

var _ HttpResult = (*httpResultWithReader)(nil)
var _ core.ResultWrapper = (*httpResultWithReader)(nil)
var _ core.ResultWithReader = (*httpResultWithReader)(nil)

func (r *httpResultWithMultipart) Unwrap() core.Result {
	return &r.httpResult
}

func (r *httpResultWithMultipart) Multipart() *multipart.Part {
	return r.multipart
}

var _ HttpResult = (*httpResultWithMultipart)(nil)
var _ core.ResultWrapper = (*httpResultWithMultipart)(nil)
var _ core.ResultWithMultipart = (*httpResultWithMultipart)(nil)
