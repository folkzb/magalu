package common

import (
	"context"

	mgcSchemaPkg "magalu.cloud/core/schema"
)

type smallFileCopier struct {
	cfg Config
	src mgcSchemaPkg.URI
	dst mgcSchemaPkg.URI
}

var _ copier = (*smallFileCopier)(nil)

func (u *smallFileCopier) Copy(ctx context.Context) error {
	req, err := newCopyRequest(ctx, u.cfg, u.src, u.dst)
	if err != nil {
		return err
	}

	resp, err := SendRequest(ctx, req)
	if err != nil {
		return err
	}

	return ExtractErr(resp, req)
}
