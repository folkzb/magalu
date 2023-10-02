package s3

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
)

type preparationResponse struct {
	XMLName  xml.Name `xml:"InitiateMultipartUploadResult"`
	Bucket   string
	Key      string
	UploadId string
}

type completionPart struct {
	Etag       string `xml:",innerxml"`
	PartNumber int
}

type completionRequest struct {
	XMLName   xml.Name         `xml:"CompleteMultipartUpload"`
	Namespace string           `xml:"xmlns,attr"`
	Parts     []completionPart `xml:"Part"`
}

type bigFileUploader struct {
	ctx      context.Context
	cfg      Config
	dst      string
	mimeType string
	readers  []io.Reader
	uploadId string
}

var _ uploader = (*bigFileUploader)(nil)

func (u *bigFileUploader) newPreparationRequest() (*http.Request, error) {
	req, err := newUploadRequest(u.ctx, u.cfg, u.dst, nil)
	if err != nil {
		return nil, err
	}
	req.Method = http.MethodPost
	req.Header.Set("Content-Type", "application/octet-stream")
	q := req.URL.Query()
	q.Set("uploads", "")
	req.URL.RawQuery = q.Encode()

	return req, nil
}

func (u *bigFileUploader) getUploadId() (string, error) {
	if u.uploadId == "" {
		req, err := u.newPreparationRequest()
		if err != nil {
			return "", err
		}

		response, _, err := SendRequest[preparationResponse](u.ctx, req, u.cfg.AccessKeyID, u.cfg.SecretKey)
		if err != nil {
			return "", err
		}
		u.uploadId = response.UploadId
	}
	return u.uploadId, nil
}

func (u *bigFileUploader) createMultipartRequest(index int, body io.Reader) (*http.Request, error) {
	uploadId, err := u.getUploadId()
	if err != nil {
		return nil, err
	}
	req, err := newUploadRequest(u.ctx, u.cfg, u.dst, body)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("uploadId", uploadId)
	q.Set("partNumber", fmt.Sprint(index+1))
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", u.mimeType)

	return req, nil
}

func (u *bigFileUploader) sendCompletionRequest(etags []string) error {
	uploadId, err := u.getUploadId()
	if err != nil {
		return err
	}

	parts := make([]completionPart, len(etags))
	for i, etag := range etags {
		// Manual string is needed here because content has double quotes.
		// Using xml.Marshal normally, the quotes are escaped, which cannot happen
		// Using `xml:",innerxml"` in struct solves this but removes tags
		parsedTag := fmt.Sprintf("<ETag>%s</ETag>", etag)
		parts[i] = completionPart{
			Etag:       parsedTag,
			PartNumber: i + 1,
		}
	}
	body := completionRequest{
		Parts:     parts,
		Namespace: "http://s3.amazonaws.com/doc/2006-03-01/",
	}
	parsed, err := xml.Marshal(body)
	if err != nil {
		return err
	}

	req, err := newUploadRequest(u.ctx, u.cfg, u.dst, bytes.NewReader(parsed))
	if err != nil {
		return err
	}
	req.Method = http.MethodPost
	q := req.URL.Query()
	q.Set("uploadId", uploadId)
	req.URL.RawQuery = q.Encode()

	// excludedHeaders is a global variable that needs to be altered specifically
	// for this request, so set the correct headers and resets after
	excludedHeaders["Content-Type"] = nil
	excludedHeaders["Content-MD5"] = nil
	defer func() {
		delete(excludedHeaders, "Content-Type")
		delete(excludedHeaders, "Content-MD5")
	}()

	_, _, err = SendRequest[any](u.ctx, req, u.cfg.AccessKeyID, u.cfg.SecretKey)
	if err != nil {
		return err
	}

	return nil
}

func (u *bigFileUploader) Upload() error {
	etags := make([]string, len(u.readers))
	for i, reader := range u.readers {
		// TODO Add retry to error handling so it doesn't block others if error
		req, err := u.createMultipartRequest(i, reader)
		if err != nil {
			return err
		}

		fmt.Printf("Sending %d of %d\n", i+1, len(u.readers))
		_, res, err := SendRequest[any](u.ctx, req, u.cfg.AccessKeyID, u.cfg.SecretKey)
		if err != nil {
			return err
		}
		etags[i] = res.Header.Get("etag")
	}
	fmt.Println("All file parts uploaded, sending completion")
	return u.sendCompletionRequest(etags)
}
