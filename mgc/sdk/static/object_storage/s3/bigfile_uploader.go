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

func NewCompletionPart(partNumber int, etag string) completionPart {
	return completionPart{
		Etag:       fmt.Sprintf("<ETag>%s</ETag>", etag),
		PartNumber: partNumber,
	}
}

type completionRequest struct {
	XMLName   xml.Name         `xml:"CompleteMultipartUpload"`
	Namespace string           `xml:"xmlns,attr"`
	Parts     []completionPart `xml:"Part"`
}

type bigFileUploader struct {
	cfg      Config
	dst      string
	mimeType string
	readers  []io.Reader
	uploadId string
}

var _ uploader = (*bigFileUploader)(nil)

func (u *bigFileUploader) newPreparationRequest(ctx context.Context) (*http.Request, error) {
	req, err := newUploadRequest(ctx, u.cfg, u.dst, nil)
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

func (u *bigFileUploader) getUploadId(ctx context.Context) (string, error) {
	if u.uploadId == "" {
		req, err := u.newPreparationRequest(ctx)
		if err != nil {
			return "", err
		}

		response, _, err := SendRequest[preparationResponse](ctx, req, u.cfg.AccessKeyID, u.cfg.SecretKey)
		if err != nil {
			return "", err
		}
		u.uploadId = response.UploadId
	}
	return u.uploadId, nil
}

func (u *bigFileUploader) createMultipartRequest(ctx context.Context, index int, body io.Reader) (*http.Request, error) {
	uploadId, err := u.getUploadId(ctx)
	if err != nil {
		return nil, err
	}
	req, err := newUploadRequest(ctx, u.cfg, u.dst, body)
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

func (u *bigFileUploader) sendCompletionRequest(ctx context.Context, parts []completionPart) error {
	uploadId, err := u.getUploadId(ctx)
	if err != nil {
		return err
	}

	body := completionRequest{
		Parts:     parts,
		Namespace: "http://s3.amazonaws.com/doc/2006-03-01/",
	}
	parsed, err := xml.Marshal(body)
	if err != nil {
		return err
	}

	req, err := newUploadRequest(ctx, u.cfg, u.dst, bytes.NewReader(parsed))
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

	_, _, err = SendRequest[any](ctx, req, u.cfg.AccessKeyID, u.cfg.SecretKey)
	if err != nil {
		return err
	}

	return nil
}

func (u *bigFileUploader) Upload(ctx context.Context) error {
	etags := make([]completionPart, len(u.readers))
	for i, reader := range u.readers {
		// TODO Add retry to error handling so it doesn't block others if error
		req, err := u.createMultipartRequest(ctx, i, reader)
		if err != nil {
			return err
		}

		fmt.Printf("Sending %d of %d\n", i+1, len(u.readers))
		_, res, err := SendRequest[any](ctx, req, u.cfg.AccessKeyID, u.cfg.SecretKey)
		if err != nil {
			return err
		}
		etags[i] = NewCompletionPart(i+1, res.Header.Get("etag"))
	}
	fmt.Println("All file parts uploaded, sending completion")
	return u.sendCompletionRequest(ctx, etags)
}
