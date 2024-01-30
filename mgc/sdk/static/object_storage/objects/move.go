package objects

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"magalu.cloud/core"
	"magalu.cloud/core/pipeline"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/object_storage/common"
)

type moveParams struct {
	Source      mgcSchemaPkg.URI `json:"src" jsonschema:"description=Source path or uri to move files from" mgc:"positional"`
	Destination mgcSchemaPkg.URI `json:"dst" jsonschema:"description=Destination to put files into" mgc:"positional"`
	BatchSize   int              `json:"batch_size,omitempty" jsonschema:"description=Limit of items per batch to process,default=1000,minimum=1,maximum=1000" example:"1000"`
}

var getMove = utils.NewLazyLoader[core.Executor](func() core.Executor {
	executor := core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "move",
			Summary:     "Moves objects from source to destination",
			Description: "Moves objects from a source to a destination.\nThey can be either local or remote but not both local (Local -> Remote, Remote -> Local, Remote -> Remote)",
		},
		move,
	)

	return core.NewExecuteResultOutputOptions(executor, func(exec core.Executor, result core.Result) string {
		return "template=Moved from {{.src}} to {{.dst}}\n"
	})
})

func move(ctx context.Context, params moveParams, cfg common.Config) (moveParams, error) {
	srcIsRemote := isRemote(params.Source)
	dstIsRemote := isRemote(params.Destination)

	if !srcIsRemote && dstIsRemote {
		return moveLocalRemote(ctx, params, cfg)
	}
	if srcIsRemote && !dstIsRemote {
		return moveRemoteLocal(ctx, params, cfg)
	}
	if srcIsRemote && dstIsRemote {
		return moveRemote(ctx, params, cfg)
	}
	if !srcIsRemote && !dstIsRemote {
		return params, core.UsageError{Err: fmt.Errorf("operation not supported, this command cannot be used to move a local source to a local destination")}
	}

	return params, nil
}

func createObjectLocalRemoteMoveProcessor(cfg common.Config, destination mgcSchemaPkg.URI, srcAbs string) pipeline.Processor[pipeline.WalkDirEntry, error] {
	return func(ctx context.Context, dirEntry pipeline.WalkDirEntry) (error, pipeline.ProcessStatus) {
		if err := dirEntry.Err(); err != nil {
			return &common.ObjectError{Err: err}, pipeline.ProcessAbort
		}

		if dirEntry.DirEntry().IsDir() {
			return nil, pipeline.ProcessOutput
		}

		absEntry, err := filepath.Abs(dirEntry.Path())
		if err != nil {
			return &common.ObjectError{Err: err}, pipeline.ProcessOutput
		}
		relative, err := filepath.Rel(srcAbs, absEntry)
		if err != nil {
			return &common.ObjectError{Err: err}, pipeline.ProcessOutput
		}
		objURI := destination.JoinPath(relative)

		_, err = upload(
			ctx,
			uploadParams{Source: mgcSchemaPkg.FilePath(dirEntry.Path()), Destination: mgcSchemaPkg.URI(objURI)},
			cfg,
		)

		if err != nil {
			return &common.ObjectError{Url: mgcSchemaPkg.URI(objURI), Err: err}, pipeline.ProcessOutput
		}

		err = os.Remove(absEntry)
		if err != nil {
			return &common.ObjectError{Url: mgcSchemaPkg.URI(objURI), Err: err}, pipeline.ProcessOutput
		}

		return nil, pipeline.ProcessOutput
	}
}

func moveRemote(ctx context.Context, params moveParams, cfg common.Config) (moveParams, error) {
	err := common.CopyMultipleFiles(ctx, cfg, common.CopyAllObjectsParams{
		Source:      params.Source,
		Destination: params.Destination,
	})
	if err != nil {
		return params, err
	}

	srcObjects := common.ListGenerator(ctx, common.ListObjectsParams{
		Destination: params.Source,
		Recursive:   true,
		PaginationParams: common.PaginationParams{
			MaxItems: common.MaxBatchSize,
		},
	}, cfg, nil)
	err = common.DeleteObjects(ctx, common.DeleteObjectsParams{
		Destination: params.Source,
		ToDelete:    srcObjects,
		BatchSize:   params.BatchSize,
	}, cfg)
	if err != nil {
		return params, err
	}

	return params, nil
}

func moveLocalRemote(ctx context.Context, params moveParams, cfg common.Config) (moveParams, error) {
	srcObjects := pipeline.WalkDirEntries(ctx, params.Source.String(), func(path string, d fs.DirEntry, err error) error {
		return err
	})
	srcAbs, err := filepath.Abs(params.Source.String())
	if err != nil {
		return params, err
	}
	srcDir := filepath.Dir(srcAbs)

	uploadObjectsErrorChan := pipeline.ParallelProcess(ctx, cfg.Workers, srcObjects, createObjectLocalRemoteMoveProcessor(cfg, params.Destination, srcDir), nil)
	uploadObjectsErrorChan = pipeline.Filter(ctx, uploadObjectsErrorChan, pipeline.FilterNonNil[error]{})

	objErr, _ := pipeline.SliceItemConsumer[utils.MultiError](ctx, uploadObjectsErrorChan)
	if len(objErr) > 0 {
		return params, objErr
	}

	return params, nil
}

func moveRemoteLocal(ctx context.Context, params moveParams, cfg common.Config) (moveParams, error) {
	_, err := downloadAll(ctx, downloadAllObjectsParams{
		Source:      params.Source,
		Destination: params.Destination.AsFilePath(),
	}, cfg)
	if err != nil {
		return params, err
	}

	srcObjects := common.ListGenerator(ctx, common.ListObjectsParams{
		Destination: params.Source,
		Recursive:   true,
		PaginationParams: common.PaginationParams{
			MaxItems: common.MaxBatchSize,
		},
	}, cfg, nil)
	err = common.DeleteObjects(ctx, common.DeleteObjectsParams{
		Destination: params.Source,
		ToDelete:    srcObjects,
		BatchSize:   params.BatchSize,
	}, cfg)
	if err != nil {
		return params, err
	}

	return params, nil
}

func isRemote(path mgcSchemaPkg.URI) bool {
	return path.Scheme() == "s3"
}
