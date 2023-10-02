package s3

import (
	"context"
	"io"
)

type smallFileUploader struct {
	ctx      context.Context
	cfg      Config
	dst      string
	mimeType string
	reader   io.Reader
}

var _ uploader = (*smallFileUploader)(nil)

func (u *smallFileUploader) Upload() error {
	req, err := newUploadRequest(u.ctx, u.cfg, u.dst, u.reader)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", u.mimeType)

	_, _, err = SendRequest[any](u.ctx, req, u.cfg.AccessKeyID, u.cfg.SecretKey)
	if err != nil {
		return err
	}
	return nil
}
