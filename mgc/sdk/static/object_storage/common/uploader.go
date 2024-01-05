package common

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"math"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
)

type uploader interface {
	Upload(context.Context) error
}

func NewUploader(cfg Config, src mgcSchemaPkg.FilePath, dst mgcSchemaPkg.URI) (uploader, error) {
	reader, fileInfo, err := readContent(src)
	if err != nil {
		return nil, fmt.Errorf("error reading object: %w", err)
	}
	size := fileInfo.Size()
	mimeType := mime.TypeByExtension(filepath.Ext(fileInfo.Name()))

	chunkN := int(math.Ceil(float64(size) / float64(CHUNK_SIZE)))

	if chunkN > 1 {
		return &bigFileUploader{
			cfg:      cfg,
			dst:      dst,
			mimeType: mimeType,
			reader:   reader,
			fileInfo: fileInfo,
			workerN:  cfg.Workers,
		}, nil
	} else {
		return &smallFileUploader{
			cfg:      cfg,
			dst:      dst,
			mimeType: mimeType,
			reader:   reader,
			fileInfo: fileInfo,
		}, nil
	}
}

func newUploadRequest(ctx context.Context, cfg Config, dst mgcSchemaPkg.URI, reader io.Reader) (*http.Request, error) {
	host, err := BuildBucketHostWithPath(cfg, NewBucketNameFromURI(dst), dst.Path())
	if err != nil {
		return nil, core.UsageError{Err: err}
	}
	return http.NewRequestWithContext(ctx, http.MethodPut, string(host), reader)
}

func readContent(p mgcSchemaPkg.FilePath) (*os.File, fs.FileInfo, error) {
	path := p.String()
	file, err := os.Stat(path)
	if err != nil {
		return nil, nil, core.UsageError{Err: err}
	}

	switch mode := file.Mode(); {
	case mode&os.ModeSymlink != 0:
		resolvedPath, err := os.Readlink(path)
		if err != nil {
			return nil, nil, err
		}
		reader, err := os.Open(resolvedPath)
		return reader, file, err
	case mode.IsRegular():
		reader, err := os.Open(path)
		return reader, file, err
	default:
		return nil, nil, fmt.Errorf("file type %s not supported", mode.Type())
	}
}
