package objects

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"magalu.cloud/core"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/common"
)

type ListObjectsParams struct {
	Destination string `json:"dst" jsonschema:"description=Path of the bucket to list objects from" example:"s3://bucket1/"`
}

type BucketContent struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	Size         int    `xml:"Size"`
}

type prefix struct {
	Path string `xml:"Prefix"`
}

type ListObjectsResponse struct {
	Name           string           `xml:"Name"`
	Contents       []*BucketContent `xml:"Contents"`
	CommonPrefixes []*prefix        `xml:"CommonPrefixes" json:"SubDirectories"`
}

func newListRequest(ctx context.Context, cfg common.Config, bucket string) (*http.Request, error) {
	parsedUrl, err := parseURL(cfg, bucket)
	if err != nil {
		return nil, err
	}
	return http.NewRequestWithContext(ctx, http.MethodGet, parsedUrl.String(), nil)
}

var getList = utils.NewLazyLoader[core.Executor](newList)

func newList() core.Executor {
	return core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "list",
			Description: "List all objects from a bucket",
		},
		List,
	)
}

func parseURL(cfg common.Config, bucketURI string) (*url.URL, error) {
	// Bucket URI cannot end in '/' as this makes it search for a
	// non existing directory
	bucketURI = strings.TrimSuffix(bucketURI, "/")
	dirs := strings.Split(bucketURI, "/")
	path, err := url.JoinPath(common.BuildHost(cfg), dirs[0])
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	if len(dirs) <= 1 {
		return u, nil
	}
	q := u.Query()
	delimiter := "/"
	prefixQ := strings.Join(dirs[1:], delimiter)
	lastChar := string(prefixQ[len(prefixQ)-1])
	if lastChar != delimiter {
		prefixQ += delimiter
	}
	q.Set("prefix", prefixQ)
	q.Set("delimiter", delimiter)
	q.Set("encoding-type", "url")
	u.RawQuery = q.Encode()
	return u, nil
}

func List(ctx context.Context, params ListObjectsParams, cfg common.Config) (result ListObjectsResponse, err error) {
	bucket, _ := strings.CutPrefix(params.Destination, common.URIPrefix)
	req, err := newListRequest(ctx, cfg, bucket)
	if err != nil {
		return
	}

	result, _, err = common.SendRequest[ListObjectsResponse](ctx, req)
	return
}
