package progress_report

import (
	"io"
)

type ReportReaderProgress func(n int, err error)

type progressReader struct {
	parent         io.Reader
	reportProgress ReportReaderProgress
}

// Wraps an io.Reader in another Reader which reports the amount of bytes read
func NewProgressReader(parent io.Reader, reportProgress ReportReaderProgress) *progressReader {
	return &progressReader{
		parent:         parent,
		reportProgress: reportProgress,
	}
}

func (pr *progressReader) Unwrap() io.Reader {
	return pr.parent
}

var _ io.ReadCloser = (*progressReader)(nil)

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.parent.Read(p)
	pr.reportProgress(n, err)
	return
}

func (pr *progressReader) Close() error {
	if closer, ok := pr.parent.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// END io.ReadCloser implementation
