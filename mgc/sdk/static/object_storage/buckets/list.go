package buckets

import (
	"context"
	"net/http"

	"magalu.cloud/core"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/common"
)

type BucketResponse struct {
	CreationDate string `xml:"CreationDate"`
	Name         string `xml:"Name"`
}

// Container for the owner's display name and ID.
type Owner struct {
	DisplayName *string `xml:"DisplayName"`
	ID          *string `type:"ID"`
}

type ListResponse struct {
	Buckets []*BucketResponse `xml:"Buckets>Bucket"`
	Owner   *Owner            `xml:"Owner"`
}

func newListRequest(ctx context.Context, cfg common.Config) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, http.MethodGet, common.BuildHost(cfg), nil)
}

var getList = utils.NewLazyLoader[core.Executor](func() core.Executor {
	return core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "list",
			Description: "List all existing Buckets",
		},
		list,
	)
})

func list(ctx context.Context, _ struct{}, cfg common.Config) (result ListResponse, err error) {
	req, err := newListRequest(ctx, cfg)
	if err != nil {
		return
	}

	result, _, err = common.SendRequest[ListResponse](ctx, req)
	return
}
