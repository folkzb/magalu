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

type DeleteObjectParams struct {
	Destination string `json:"dst" jsonschema:"description=Path of the object to be deleted" example:"s3://bucket1/file1"`
}

var getDelete = utils.NewLazyLoader[core.Executor](newDelete)

func newDelete() core.Executor {
	exec := core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "delete",
			Description: "Delete an object from a bucket",
		},
		Delete,
	)

	msg := "This command will delete the object at {{.parameters.dst}}, and it's result is NOT reversible."

	return core.NewConfirmableExecutor(
		exec,
		core.ConfirmPromptWithTemplate(msg),
	)
}

func newDeleteRequest(ctx context.Context, cfg common.Config, pathURIs ...string) (*http.Request, error) {
	host := common.BuildHost(cfg)
	url, err := url.JoinPath(host, pathURIs...)
	if err != nil {
		return nil, err
	}
	return http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
}

func Delete(ctx context.Context, params DeleteObjectParams, cfg common.Config) (result core.Value, err error) {
	bucketURI, _ := strings.CutPrefix(params.Destination, common.URIPrefix)
	req, err := newDeleteRequest(ctx, cfg, bucketURI)
	if err != nil {
		return nil, err
	}

	result, _, err = common.SendRequest[core.Value](ctx, req)
	return
}
