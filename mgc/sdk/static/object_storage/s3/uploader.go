package s3

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"math"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type uploader interface {
	Upload(context.Context) error
}

func NewS3Uploader(cfg Config, src, dst string) (uploader, error) {
	reader, fileInfo, err := readContent(src)
	if err != nil {
		return nil, fmt.Errorf("error reading object: %w", err)
	}
	size := fileInfo.Size()
	mimeType := mime.TypeByExtension(filepath.Ext(fileInfo.Name()))

	chunkN := int(math.Ceil(float64(size) / float64(CHUNK_SIZE)))

	if chunkN > 1 {
		readers := splitReader(reader, CHUNK_SIZE, chunkN)
		return &bigFileUploader{
			cfg:      cfg,
			dst:      dst,
			mimeType: mimeType,
			readers:  readers,
		}, nil
	} else {
		return &smallFileUploader{
			cfg:      cfg,
			dst:      dst,
			mimeType: mimeType,
			reader:   reader,
		}, nil
	}
}

func newUploadRequest(ctx context.Context, cfg Config, dst string, reader io.Reader) (*http.Request, error) {
	host := BuildHost(cfg)
	url, err := url.JoinPath(host, dst)
	if err != nil {
		return nil, err
	}
	return http.NewRequestWithContext(ctx, http.MethodPut, url, reader)
}

func splitReader(reader io.ReaderAt, chunkSize, chunkN int) []io.Reader {
	readers := make([]io.Reader, chunkN)
	for i := range readers {
		readers[i] = io.NewSectionReader(reader, int64(i*chunkSize), int64(chunkSize))
	}
	return readers
}

func readContent(path string) (*os.File, fs.FileInfo, error) {
	file, err := os.Stat(path)
	if err != nil {
		return nil, nil, err
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
		// TODO: treat directory recursively
		return nil, nil, fmt.Errorf("file type %s not supported", mode.Type())
	}
}
