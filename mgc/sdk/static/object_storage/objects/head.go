package objects

import (
	"context"

	"magalu.cloud/core"
	"magalu.cloud/core/progress_report"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/common"
)

type headObjectParams struct {
	Destination mgcSchemaPkg.URI `json:"dst" jsonschema:"description=Path of the object to be get metadata from,example=bucket1/file.txt" mgc:"positional"`
}

var getHead = utils.NewLazyLoader[core.Executor](func() core.Executor {
	return core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "head",
			Description: "Get object metadata",
		},
		headObject,
	)
})

func headObject(ctx context.Context, p headObjectParams, cfg common.Config) (result core.Value, err error) {
	reportProgress := progress_report.FromContext(ctx)
	reportMsg := "Getting metadata for " + p.Destination.String()
	progress := uint64(0)
	total := uint64(1)

	reportProgress(reportMsg, progress, progress, progress_report.UnitsNone, nil)

	result, err = common.HeadFile(ctx, cfg, p.Destination)
	if err != nil {
		reportProgress(reportMsg, progress, progress, progress_report.UnitsNone, err)
		return nil, err
	}

	reportProgress(reportMsg, total, total, progress_report.UnitsNone, progress_report.ErrorProgressDone)

	return result, nil
}
