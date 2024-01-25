package common

import (
	"context"
	"os"
	"path"

	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
)

type smallFileDownloader struct {
	cfg Config
	src mgcSchemaPkg.URI
	dst mgcSchemaPkg.FilePath
}

var _ downloader = (*smallFileDownloader)(nil)

func (u *smallFileDownloader) Download(ctx context.Context) error {
	req, err := NewDownloadRequest(ctx, u.cfg, u.src)
	if err != nil {
		return err
	}

	resp, err := SendRequest(ctx, req)
	if err != nil {
		return err
	}

	err = ExtractErr(resp)
	if err != nil {
		return err
	}

	dir := path.Dir(u.dst.String())
	if len(dir) != 0 {
		if err := os.MkdirAll(dir, utils.DIR_PERMISSION); err != nil {
			return err
		}
	}

	if err := WriteToFile(ctx, resp.Body, resp.ContentLength, u.dst); err != nil {
		return err
	}

	return nil
}
