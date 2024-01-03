package common

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
)

type DownloadObjectParams struct {
	Source      mgcSchemaPkg.URI      `json:"src" jsonschema:"description=Path of the object to be downloaded,example=s3://bucket1/file1" mgc:"positional"`
	Destination mgcSchemaPkg.FilePath `json:"dst,omitempty" jsonschema:"description=Name of the file to be saved,example=file1.txt" mgc:"positional"`
}

func NewDownloadRequest(ctx context.Context, cfg Config, src mgcSchemaPkg.URI) (*http.Request, error) {
	host, err := BuildBucketHostWithPath(cfg, NewBucketNameFromURI(src), src.Path())
	if err != nil {
		return nil, core.UsageError{Err: err}
	}
	return http.NewRequestWithContext(ctx, http.MethodGet, string(host), nil)
}

func WriteToFile(reader io.ReadCloser, outFile mgcSchemaPkg.FilePath) (err error) {
	defer reader.Close()

	writer, err := os.OpenFile(outFile.String(), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, utils.FILE_PERMISSION)
	if err != nil {
		return err
	}

	n, err := writer.ReadFrom(reader)
	defer writer.Close()
	if err != nil {
		return fmt.Errorf("error writing to file (wrote %d bytes): %w", n, err)
	}
	return nil
}

func GetDestination(dst mgcSchemaPkg.FilePath, src mgcSchemaPkg.URI) (mgcSchemaPkg.FilePath, error) {
	if dst == "" {
		d, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("no destination specified and could not use local dir: %w", err)
		}
		_, fname := path.Split(src.Path())
		return mgcSchemaPkg.FilePath(path.Join(d, fname)), nil
	}
	return dst, nil
}
