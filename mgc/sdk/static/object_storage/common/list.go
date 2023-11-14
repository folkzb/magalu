package common

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"go.uber.org/zap"
	"magalu.cloud/core/pipeline"
	"magalu.cloud/core/utils"
)

var listObjectsLogger = utils.NewLazyLoader(func() *zap.SugaredLogger {
	return logger().Named("list")
})

type ListObjectsParams struct {
	Destination      string           `json:"dst" jsonschema:"description=Path of the bucket to list objects from" example:"s3://bucket1/"`
	PaginationParams `json:",squash"` // nolint
	Recursive        bool             `json:"recursive,omitempty" jsonschema:"description=List folders and subfolders,default=false"`
	Include          string           `json:"include,omitempty" jsonschema:"description=Filename pattern to include,default=*"`
	Exclude          string           `json:"exclude,omitempty" jsonschema:"description=Filename pattern to exclude"`
}

type PaginationParams struct {
	MaxItems          int    `json:"max-items,omitempty" jsonschema:"description=Limit of items to be listed,default=1000,minimum=1" example:"1000"`
	ContinuationToken string `json:"continuation-token,omitempty" jsonschema:"description=Token of result page to continue from"`
}

type Prefix struct {
	Path string `xml:"Prefix"`
}

func (p *Prefix) ModTime() time.Time {
	var modTime time.Time
	return modTime
}

func (p *Prefix) Mode() fs.FileMode {
	return utils.DIR_PERMISSION | fs.ModeDir
}

func (p *Prefix) Size() int64 {
	return 0
}

func (p *Prefix) Sys() any {
	return nil
}

func (p *Prefix) Info() (fs.FileInfo, error) {
	return p, nil
}

func (p *Prefix) IsDir() bool {
	return true
}

func (p *Prefix) Name() string {
	return path.Base(path.Dir(p.Path))
}

func (p *Prefix) Type() fs.FileMode {
	return utils.DIR_PERMISSION | fs.ModeDir
}

var _ fs.DirEntry = (*Prefix)(nil)
var _ fs.FileInfo = (*Prefix)(nil)

type listObjectsRequestResponse struct {
	Name                   string           `xml:"Name"`
	Contents               []*BucketContent `xml:"Contents"`
	CommonPrefixes         []*Prefix        `xml:"CommonPrefixes" json:"SubDirectories"`
	paginationResponseInfo `json:",squash"` // nolint
}

type paginationResponseInfo struct {
	NextContinuationToken string `xml:"NextContinuationToken"`
	IsTruncated           bool   `xml:"IsTruncated"`
}

type BucketContent struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	ContentSize  int64  `xml:"Size"`
}

type BucketContentDirEntry = *pipeline.SimpleWalkDirEntry[*BucketContent]

func (b *BucketContent) ModTime() time.Time {
	modTime, err := time.Parse(time.RFC3339, b.LastModified)
	if err != nil {
		listObjectsLogger().Named("BucketContent.ModTime()").Errorw("failed to parse time", "err", err, "key", b.Key, "lastModified", b.LastModified)
		modTime = time.Time{}
	}
	return modTime
}

func (b *BucketContent) Mode() fs.FileMode {
	return utils.FILE_PERMISSION
}

func (b *BucketContent) Size() int64 {
	return b.ContentSize
}

func (b *BucketContent) Sys() any {
	return nil
}

func (b *BucketContent) Info() (fs.FileInfo, error) {
	return b, nil
}

func (b *BucketContent) IsDir() bool {
	return false
}

func (b *BucketContent) Name() string {
	return path.Base(b.Key)
}

func (b *BucketContent) Type() fs.FileMode {
	return utils.FILE_PERMISSION
}

var _ fs.DirEntry = (*BucketContent)(nil)
var _ fs.FileInfo = (*BucketContent)(nil)

func newListRequest(ctx context.Context, cfg Config, bucket string, page PaginationParams, recursive bool) (*http.Request, error) {
	parsedUrl, err := parseURL(cfg, bucket)
	if err != nil {
		return nil, err
	}

	listReqQuery := parsedUrl.Query()
	listReqQuery.Set("list-type", "2")
	if page.ContinuationToken != "" {
		listReqQuery.Set("continuation-token", page.ContinuationToken)
	}
	if page.MaxItems <= 0 {
		return nil, fmt.Errorf("invalid item limit MaxItems, must be higher than zero: %d", page.MaxItems)
	} else if page.MaxItems > ApiLimitMaxItems {
		page.MaxItems = ApiLimitMaxItems
	}
	listReqQuery.Set("max-keys", fmt.Sprint(page.MaxItems))
	if !recursive {
		listReqQuery.Set("delimiter", delimiter)
	}
	parsedUrl.RawQuery = listReqQuery.Encode()

	return http.NewRequestWithContext(ctx, http.MethodGet, parsedUrl.String(), nil)
}

func parseURL(cfg Config, bucketURI string) (*url.URL, error) {
	// Bucket URI cannot end in '/' as this makes it search for a
	// non existing directory
	bucketURI = strings.TrimSuffix(bucketURI, delimiter)
	pathEntries := strings.Split(bucketURI, delimiter)
	bucket := pathEntries[0]
	path, err := url.JoinPath(BuildHost(cfg), bucket)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if len(pathEntries) > 1 {
		prefixQ := strings.Join(pathEntries[1:], delimiter)
		lastChar := string(prefixQ[len(prefixQ)-1])
		if lastChar != delimiter {
			prefixQ += delimiter
		}
		q.Set("prefix", prefixQ)
	}
	q.Set("encoding-type", "url")
	u.RawQuery = q.Encode()
	return u, nil
}

func ListGenerator(ctx context.Context, params ListObjectsParams, cfg Config) (outputChan <-chan pipeline.WalkDirEntry) {
	ch := make(chan pipeline.WalkDirEntry)
	outputChan = ch

	logger := listObjectsLogger().Named("ListGenerator").With(
		"params", params,
		"cfg", cfg,
	)

	generator := func() {
		defer func() {
			close(ch)
			logger.Info("closed output channel")
		}()

		page := params.PaginationParams
		var requestedItems int
		bucket, _ := strings.CutPrefix(params.Destination, URIPrefix)
	MainLoop:
		for {
			requestedItems = 0

			req, err := newListRequest(ctx, cfg, bucket, page, params.Recursive)
			var result listObjectsRequestResponse
			if err == nil {
				result, _, err = SendRequest[listObjectsRequestResponse](ctx, req)
			}

			if err != nil {
				logger.Errorw("list request failed", "err", err, "req", req)
				select {
				case <-ctx.Done():
					logger.Debugw("context.Done()", "err", err)
				case ch <- pipeline.NewSimpleWalkDirEntry[*BucketContent](params.Destination, nil, err):
				}
				return
			}

			for _, prefix := range result.CommonPrefixes {
				dirEntry := pipeline.NewSimpleWalkDirEntry(
					path.Join(params.Destination, prefix.Path),
					prefix,
					nil,
				)

				select {
				case <-ctx.Done():
					logger.Debugw("context.Done()", "err", ctx.Err())
					return
				case ch <- dirEntry:
					requestedItems++
					if requestedItems >= page.MaxItems {
						logger.Infow("item limit reached", "limit", params.PaginationParams.MaxItems)
						break MainLoop
					}
				}
			}

			for _, content := range result.Contents {
				dirEntry := pipeline.NewSimpleWalkDirEntry(
					path.Join(params.Destination, content.Key),
					content,
					nil,
				)

				select {
				case <-ctx.Done():
					logger.Debugw("context.Done()", "err", ctx.Err())
					return
				case ch <- dirEntry:
					requestedItems++
					if requestedItems >= page.MaxItems {
						logger.Infow("item limit reached", "limit", params.PaginationParams.MaxItems)
						break MainLoop
					}
				}
			}

			page.ContinuationToken = result.NextContinuationToken
			page.MaxItems = page.MaxItems - requestedItems
			if !result.IsTruncated || page.MaxItems <= 0 {
				logger.Info("finished reading contents")
				break
			}
		}
	}

	logger.Info("list generation start")
	go generator()
	return
}
