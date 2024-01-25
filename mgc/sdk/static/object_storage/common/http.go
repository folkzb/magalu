package common

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"

	mgcHttpPkg "magalu.cloud/core/http"
	"magalu.cloud/core/utils"
	"magalu.cloud/core/xml"
)

type XMLError struct {
	Message string `xml:"Message"`
	Code    string `xml:"Code"`
}

func UnwrapResponse[T any](resp *http.Response) (result T, err error) {
	if err = ExtractErr(resp); err != nil {
		return
	}

	if resp.StatusCode == 204 {
		return
	}

	contentType, params, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))

	switch {
	default:
		err = utils.AssignToT(&result, resp.Body)
		return
	case strings.HasPrefix(contentType, "multipart/"):
		body, bodyErr := mgcHttpPkg.BodyReaderSafe(resp)
		if bodyErr != nil {
			err = fmt.Errorf("error when reading response body: %w", bodyErr)
			return
		}
		// TODO: do we have multi-part downloads? or just single?
		// If multi, then we need to return a multipart reader...
		// return multipart.NewReader(resp.Body, params["boundary"]), nil
		r := multipart.NewReader(body, params["boundary"])
		nextPart, npErr := r.NextPart()
		err = npErr
		if err != nil {
			return
		}
		err = utils.AssignToT(&result, nextPart)
		return
	case contentType == "application/json":
		err = mgcHttpPkg.DecodeJSON(resp, &result)
	case contentType == "application/xml":
		err = mgcHttpPkg.DecodeXML(resp, &result)
	}

	return
}

func ExtractErr(resp *http.Response) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newHttpError(resp)
	}
	return nil
}

func newHttpError(resp *http.Response) *mgcHttpPkg.HttpError {
	slug := "unknown"
	message := resp.Status

	defer resp.Body.Close()

	payload, _ := io.ReadAll(resp.Body)
	contentType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		logger().Debugw("ignored invalid response", "Content-Type", resp.Header.Get("Content-Type"), "error", err.Error())
	}

	if contentType == "application/json" {
		// Although the contentType is application/json, in the object-storage specific case,
		// the response comes in XML format. Thus we have to decode it as such
		data := XMLError{}
		decoder := xml.NewDecoder(bytes.NewBuffer(payload))

		if err := decoder.Decode(&data); err == nil {
			// fmt.Printf("%#v\n", data)
			if data.Message != "" {
				message = data.Message
			}
			if data.Code != "" {
				slug = data.Code
			}
		}
	}

	return &mgcHttpPkg.HttpError{
		Code:    resp.StatusCode,
		Status:  resp.Status,
		Headers: resp.Header,
		Payload: payload,
		Message: message,
		Slug:    slug,
	}
}