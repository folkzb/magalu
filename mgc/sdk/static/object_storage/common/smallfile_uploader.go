package common

import (
	"context"
	"fmt"
	"io"
	"io/fs"

	"magalu.cloud/core/progress_report"
	mgcSchemaPkg "magalu.cloud/core/schema"
)

type smallFileUploader struct {
	cfg      Config
	dst      mgcSchemaPkg.URI
	mimeType string
	fileInfo fs.FileInfo
	filePath mgcSchemaPkg.FilePath
}

var _ uploader = (*smallFileUploader)(nil)

func (u *smallFileUploader) Upload(ctx context.Context) error {
	progressReporter := progress_report.NewBytesReporter(ctx, "Uploading "+u.fileInfo.Name(), uint64(u.fileInfo.Size()))
	progressReporter.Start()
	defer progressReporter.End()

	ctx = progress_report.NewBytesReporterContext(ctx, progressReporter)
	newReader := func() (io.ReadCloser, error) {
		reader, err := readContent(u.filePath, u.fileInfo)
		if err != nil {
			return nil, fmt.Errorf("error reading file: %w", err)
		}
		return reader, nil
	}

	req, err := newUploadRequest(ctx, u.cfg, u.dst, newReader)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", u.mimeType)

	resp, err := SendRequest(ctx, req)
	if err != nil {
		return err
	}

	return ExtractErr(resp, req)
}
