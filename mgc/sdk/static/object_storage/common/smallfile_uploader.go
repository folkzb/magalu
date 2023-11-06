package common

import (
	"context"
	"io"
	"io/fs"

	"magalu.cloud/core/progress_report"
)

type smallFileUploader struct {
	cfg      Config
	dst      string
	mimeType string
	reader   io.Reader
	fileInfo fs.FileInfo
}

var _ uploader = (*smallFileUploader)(nil)

func (u *smallFileUploader) createProgressReporter(ctx context.Context) progress_report.ReportReaderProgress {
	reportProgress := progress_report.FromContext(ctx)
	fileName := u.fileInfo.Name()
	total := uint64(u.fileInfo.Size())
	sentBytes := uint64(0)
	return func(n int, err error) {
		sentBytes += uint64(n)
		reportProgress(fileName, sentBytes, total, progress_report.UnitsBytes, err)
	}
}

func (u *smallFileUploader) Upload(ctx context.Context) error {
	wrappedReader := progress_report.NewProgressReader(u.reader, u.createProgressReporter(ctx))
	req, err := newUploadRequest(ctx, u.cfg, u.dst, wrappedReader)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", u.mimeType)

	_, _, err = SendRequest[any](ctx, req)
	if err != nil {
		return err
	}
	return nil
}