package common

import (
	"context"
	"net/http"
	"strconv"

	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
)

type headObjectRequestResponse struct {
	AcceptRanges  string
	LastModified  string
	ContentLength int64
	ETag          string
	ContentType   string
}

func newHeadRequest(ctx context.Context, cfg Config, dst mgcSchemaPkg.URI) (*http.Request, error) {
	host, err := BuildBucketHostWithPath(cfg, NewBucketNameFromURI(dst), dst.Path())
	if err != nil {
		return nil, core.UsageError{Err: err}
	}
	return http.NewRequestWithContext(ctx, http.MethodHead, string(host), nil)
}

func HeadFile(ctx context.Context, cfg Config, dst mgcSchemaPkg.URI) (metadata headObjectRequestResponse, err error) {
	req, err := newHeadRequest(ctx, cfg, dst)
	if err != nil {
		return
	}

	resp, err := SendRequest(ctx, req)
	if err != nil {
		return
	}

	err = ExtractErr(resp)
	if err != nil {
		return
	}

	metadata, err = getMetadataFromResponse(resp)
	if err != nil {
		return
	}

	return
}

func parseContentLength(contentLenght string) (int64, error) {
	value, err := strconv.ParseInt(contentLenght, HeadContentLengthBase, HeadContentLengthBitSize)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func getMetadataFromResponse(resp *http.Response) (headObjectRequestResponse, error) {
	contentLength, err := parseContentLength(resp.Header.Get("Content-Length"))
	if err != nil {
		return headObjectRequestResponse{}, err
	}

	metadata := headObjectRequestResponse{
		AcceptRanges:  resp.Header.Get("Accept-Ranges"),
		LastModified:  resp.Header.Get("Last-Modified"),
		ContentLength: contentLength,
		ETag:          resp.Header.Get("ETag"),
		ContentType:   resp.Header.Get("Content-Type"),
	}

	return metadata, nil
}
