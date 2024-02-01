package common

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"magalu.cloud/core"
	"magalu.cloud/core/progress_report"
	mgcSchemaPkg "magalu.cloud/core/schema"
)

type CopyObjectParams struct {
	Source      mgcSchemaPkg.URI `json:"src" jsonschema:"description=Path of the object to be copied,example=s3://bucket1/file.txt" mgc:"positional"`
	Destination mgcSchemaPkg.URI `json:"dst" jsonschema:"description=Full destination path in the bucket with desired filename,example=s3://bucket2/dir/file.txt" mgc:"positional"`
}

func newCopyRequest(ctx context.Context, cfg Config, src mgcSchemaPkg.URI, dst mgcSchemaPkg.URI) (*http.Request, error) {
	host, err := BuildBucketHostWithPath(cfg, NewBucketNameFromURI(dst), dst.Path())
	if err != nil {
		return nil, core.UsageError{Err: err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, string(host), nil)
	if err != nil {
		return nil, err
	}

	copySource, err := url.JoinPath(src.Hostname(), src.Path())
	if err != nil {
		return nil, core.UsageError{Err: fmt.Errorf("badly specified source URI: %w", err)}
	}

	req.Header.Set("x-amz-copy-source", copySource)

	return req, nil
}

func CopySingleFile(ctx context.Context, cfg Config, src mgcSchemaPkg.URI, dst mgcSchemaPkg.URI) error {
	reportProgress := progress_report.FromContext(ctx)
	reportMsg := "Copying object from " + src.String() + " to " + dst.String()
	progress := uint64(0)
	total := uint64(1)

	reportProgress(reportMsg, progress, progress, progress_report.UnitsNone, nil)

	req, err := newCopyRequest(ctx, cfg, src, dst)
	if err != nil {
		reportProgress(reportMsg, progress, progress, progress_report.UnitsNone, err)
		return err
	}

	resp, err := SendRequest(ctx, req)
	if err != nil {
		reportProgress(reportMsg, progress, progress, progress_report.UnitsNone, err)
		return err
	}

	reportProgress(reportMsg, total, total, progress_report.UnitsNone, progress_report.ErrorProgressDone)

	return ExtractErr(resp)
}
