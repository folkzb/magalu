package objects

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"

	"go.uber.org/zap"
	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/common"
)

var downloadObjectsLogger *zap.SugaredLogger

func downloadLogger() *zap.SugaredLogger {
	if downloadObjectsLogger == nil {
		downloadObjectsLogger = logger().Named("download")
	}
	return downloadObjectsLogger
}

type downloadObjectParams struct {
	Source                  mgcSchemaPkg.URI      `json:"src" jsonschema:"description=Path of the object to be downloaded,example=s3://bucket1/file1" mgc:"positional"`
	Destination             mgcSchemaPkg.FilePath `json:"dst,omitempty" jsonschema:"description=Name of the file to be saved,example=file1.txt" mgc:"positional"`
	common.PaginationParams `json:",squash"`      // nolint
}

var getDownload = utils.NewLazyLoader[core.Executor](func() core.Executor {
	executor := core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "download",
			Description: "download an object from a bucket",
		},
		download,
	)

	return core.NewExecuteResultOutputOptions(executor, func(exec core.Executor, result core.Result) string {
		return "template=Downloaded from {{.src}} to {{.dst}}\n"
	})
})

func newDownloadRequest(ctx context.Context, cfg common.Config, src mgcSchemaPkg.URI) (*http.Request, error) {
	host := common.BuildHost(cfg)
	url, err := url.JoinPath(host, src.Path())
	if err != nil {
		return nil, core.UsageError{Err: err}
	}
	return http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
}

func writeToFile(reader io.ReadCloser, outFile mgcSchemaPkg.FilePath) (err error) {
	defer reader.Close()

	writer, err := os.OpenFile(outFile.String(), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
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

func downloadSingleFile(ctx context.Context, cfg common.Config, src mgcSchemaPkg.URI, dst mgcSchemaPkg.FilePath) error {
	req, err := newDownloadRequest(ctx, cfg, src)
	if err != nil {
		return err
	}

	closer, _, err := common.SendRequest[io.ReadCloser](ctx, req)
	if err != nil {
		return err
	}

	dir := path.Dir(dst.String())
	if len(dir) != 0 {
		if err := os.MkdirAll(dir, utils.DIR_PERMISSION); err != nil {
			return err
		}
	}

	if err := writeToFile(closer, dst); err != nil {
		return err
	}

	return nil
}

func downloadMultipleFiles(ctx context.Context, cfg common.Config, src mgcSchemaPkg.URI, dst mgcSchemaPkg.FilePath, paginationParams common.PaginationParams) error {
	listParams := common.ListObjectsParams{
		Destination:      src,
		Recursive:        true,
		PaginationParams: paginationParams,
	}
	dirEntries := common.ListGenerator(ctx, listParams, cfg)

	bucketName := common.NewBucketNameFromURI(src)
	rootURI := bucketName.AsURI()
	var errors utils.MultiError
	for dirEntry := range dirEntries {
		objURI := rootURI.JoinPath(dirEntry.Path())

		if err := dirEntry.Err(); err != nil {
			errors = append(errors, &common.ObjectError{Url: objURI, Err: err})
			continue
		}

		obj, ok := dirEntry.DirEntry().(*common.BucketContent)
		if !ok {
			errors = append(errors, &common.ObjectError{Url: objURI, Err: fmt.Errorf("expected object, got directory")})
			continue
		}

		downloadLogger().Infow("Downloading object", "uri", objURI)
		// TODO: change API to use BucketName, URI and FilePath
		req, err := newDownloadRequest(ctx, cfg, mgcSchemaPkg.URI(objURI))
		if err != nil {

			errors = append(errors, &common.ObjectError{Url: objURI, Err: err})
			continue
		}

		closer, _, err := common.SendRequest[io.ReadCloser](ctx, req)
		if err != nil || closer == nil {
			errors = append(errors, &common.ObjectError{Url: objURI, Err: err})
			continue
		}

		dir := path.Dir(obj.Key)
		if err := os.MkdirAll(path.Join(dst.String(), dir), utils.DIR_PERMISSION); err != nil {
			errors = append(errors, &common.ObjectError{Url: objURI, Err: err})
			continue
		}

		if err := writeToFile(closer, dst.Join(obj.Key)); err != nil {
			errors = append(errors, &common.ObjectError{Url: objURI, Err: err})
			continue
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

func newHeadObjectRequest(ctx context.Context, cfg common.Config, src mgcSchemaPkg.URI) (*http.Request, error) {
	host := common.BuildHost(cfg)
	url, err := url.JoinPath(host, src.Path())
	if err != nil {
		return nil, core.UsageError{Err: err}
	}
	return http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
}

func isObjectPath(ctx context.Context, cfg common.Config, src mgcSchemaPkg.URI) bool {
	req, err := newHeadObjectRequest(ctx, cfg, src)
	if err != nil {
		return false
	}

	result, _, err := common.SendRequest[core.Value](ctx, req)
	if err != nil {
		return false
	}

	return result != nil
}

func isDirPath(fpath mgcSchemaPkg.FilePath) bool {
	return path.Ext(fpath.String()) == ""
}

func download(ctx context.Context, p downloadObjectParams, cfg common.Config) (result core.Value, err error) {
	dst := p.Destination
	if dst == "" {
		var d string
		d, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("no destination specified and could not use local dir: %w", err)
		}
		fname := path.Base(p.Source.Path())
		dst = mgcSchemaPkg.FilePath(path.Join(d, fname))
	}
	src := p.Source
	if isObjectPath(ctx, cfg, src) {
		// User specified a directory, append the file name to it
		if isDirPath(dst) {
			fname := path.Base(src.Path())
			dst = dst.Join(fname)
		}
		err = downloadSingleFile(ctx, cfg, src, dst)
	} else {
		if !isDirPath(dst) {
			return nil, fmt.Errorf("bucket resource %s is a directory but given local path is a file %s", p.Source, p.Destination)
		}
		p.MaxItems = math.MaxInt64
		err = downloadMultipleFiles(ctx, cfg, src, dst, p.PaginationParams)
	}

	if err != nil {
		return nil, err
	}

	// TODO: change API to use BucketName, URI and FilePath
	return downloadObjectParams{Source: p.Source, Destination: mgcSchemaPkg.FilePath(dst)}, nil
}
