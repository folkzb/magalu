package common

import (
	"context"
	"fmt"
	"io"
	"os"

	"magalu.cloud/core/pipeline"
	"magalu.cloud/core/progress_report"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
)

type downloadPartsProgressReport struct {
	bytes uint64
	err   error
}

type bigFileDownloader struct {
	cfg        Config
	src        mgcSchemaPkg.URI
	dst        mgcSchemaPkg.FilePath
	fileSize   int64
	reportChan chan downloadPartsProgressReport
}

func (u *bigFileDownloader) createPartDownloaderProcessor(cancel context.CancelCauseFunc, cfg Config) pipeline.Processor[pipeline.WriteableChunk, error] {
	return func(ctx context.Context, chunk pipeline.WriteableChunk) (error, pipeline.ProcessStatus) {
		req, err := NewDownloadRequest(ctx, cfg, u.src)
		if err != nil {
			cancel(err)
			return err, pipeline.ProcessAbort
		}

		downloadByteRange := fmt.Sprintf("bytes=%d-%d", chunk.StartOffset, chunk.EndOffset)
		req.Header.Set("Range", downloadByteRange)

		resp, err := SendRequest(ctx, req)
		if err != nil {
			cancel(err)
			return err, pipeline.ProcessAbort
		}

		err = ExtractErr(resp)
		if err != nil {
			cancel(err)
			return err, pipeline.ProcessAbort
		}

		reporterWriter := progress_report.NewReporterWriter(chunk.Writer, func(n int, err error) {
			u.reportChan <- downloadPartsProgressReport{bytes: uint64(n), err: err}
		})

		_, err = io.Copy(reporterWriter, resp.Body)
		if err != nil {
			return err, pipeline.ProcessAbort
		}

		return nil, pipeline.ProcessOutput
	}
}

func (u *bigFileDownloader) Download(ctx context.Context) error {
	reportProgress := progress_report.FromContext(ctx)
	u.reportChan = make(chan downloadPartsProgressReport)
	defer close(u.reportChan)

	go downloadPartsProgressReportSubroutine(reportProgress, u.reportChan, u.src.String(), u.fileSize)

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	writer, err := os.OpenFile(u.dst.String(), os.O_WRONLY|os.O_CREATE, utils.FILE_PERMISSION)
	if err != nil {
		return err
	}

	chunkChan := pipeline.PrepareWriteChunks(ctx, writer, u.fileSize, CHUNK_SIZE)

	bigFileDownloadErrorChan := pipeline.ParallelProcess(ctx, u.cfg.Workers, chunkChan, u.createPartDownloaderProcessor(cancel, u.cfg), nil)
	bigFileDownloadErrorChan = pipeline.Filter(ctx, bigFileDownloadErrorChan, pipeline.FilterNonNil[error]{})

	objErr, _ := pipeline.SliceItemConsumer[utils.MultiError](ctx, bigFileDownloadErrorChan)
	if len(objErr) > 0 {
		return objErr
	}

	return nil
}

func downloadPartsProgressReportSubroutine(
	reportProgress progress_report.ReportProgress,
	reportChan <-chan downloadPartsProgressReport,
	name string,
	contentLength int64,
) {
	total := uint64(contentLength)
	bytesDone := uint64(0)

	reportProgress(name, bytesDone, total, progress_report.UnitsBytes, nil)

	var err error

	for report := range reportChan {
		bytesDone += report.bytes
		if report.err != nil {
			err = report.err
		}
		reportProgress(name, bytesDone, total, progress_report.UnitsBytes, nil)
	}
	if err == nil {
		err = progress_report.ErrorProgressDone
	}

	reportProgress(name, bytesDone, total, progress_report.UnitsBytes, err)
}
