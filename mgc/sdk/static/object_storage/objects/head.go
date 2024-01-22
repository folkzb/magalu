package objects

import (
	"context"
	"net/http"
	"strconv"

	"magalu.cloud/core"
	"magalu.cloud/core/progress_report"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/common"
)

type headObjectParams struct {
	Destination mgcSchemaPkg.URI `json:"dst" jsonschema:"description=Path of the object to be get metadata from,example=s3://bucket1/file.txt" mgc:"positional"`
}

type headObjectRequestResponse struct {
	AcceptRanges  string
	LastModified  string
	ContentLength int64
	ETag          string
	ContentType   string
}

var getHead = utils.NewLazyLoader[core.Executor](func() core.Executor {
	return core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "head",
			Description: "Get object metadata",
		},
		headObject,
	)
})

func newHeadRequest(ctx context.Context, cfg common.Config, dst mgcSchemaPkg.URI) (*http.Request, error) {
	host, err := common.BuildBucketHostWithPath(cfg, common.NewBucketNameFromURI(dst), dst.Path())
	if err != nil {
		return nil, core.UsageError{Err: err}
	}
	return http.NewRequestWithContext(ctx, http.MethodHead, string(host), nil)
}

func headFile(ctx context.Context, cfg common.Config, dst mgcSchemaPkg.URI) (*http.Response, error) {
	req, err := newHeadRequest(ctx, cfg, dst)
	if err != nil {
		return nil, err
	}

	resp, err := common.SendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	err = common.ExtractErr(resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func parseContentLength(contentLenght string) (int64, error) {
	value, err := strconv.ParseInt(contentLenght, common.HeadContentLengthBase, common.HeadContentLengthBitSize)
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

func headObject(ctx context.Context, p headObjectParams, cfg common.Config) (result core.Value, err error) {
	reportProgress := progress_report.FromContext(ctx)
	reportMsg := "Getting metadata for " + p.Destination.String()
	progress := uint64(0)
	total := uint64(1)

	reportProgress(reportMsg, progress, progress, progress_report.UnitsNone, nil)

	resp, err := headFile(ctx, cfg, p.Destination)
	if err != nil {
		reportProgress(reportMsg, progress, progress, progress_report.UnitsNone, err)
		return nil, err
	}

	reportProgress(reportMsg, total, total, progress_report.UnitsNone, progress_report.ErrorProgressDone)

	result, err = getMetadataFromResponse(resp)
	if err != nil {
		return nil, err
	}

	return result, nil
}
