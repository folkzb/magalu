package logger

import (
	"io"
)

type LogReadData struct {
	Size int    `json:"size"`
	Data string `json:"data"`
	Err  error  `json:"error,omitempty"`
}

type progressiveLoggerReader struct {
	parent io.Reader
	logger func(readData LogReadData)
}

// Wraps an io.Reader in another Reader which progressively logs the read data as the parent is read
func NewProgressiveLoggerReader(parent io.Reader, logger func(readData LogReadData)) io.ReadCloser {
	return &progressiveLoggerReader{
		parent: parent,
		logger: logger,
	}
}

func (pr *progressiveLoggerReader) Unwrap() io.Reader {
	return pr.parent
}

// BEGIN io.ReadCloser implementation

func (pr *progressiveLoggerReader) Read(p []byte) (n int, err error) {
	n, err = pr.parent.Read(p)
	pr.logger(LogReadData{Size: n, Data: string(p[:n]), Err: err})
	return
}

func (pr *progressiveLoggerReader) Close() error {
	if closer, ok := pr.parent.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

var _ io.ReadCloser = (*progressiveLoggerReader)(nil)

// END io.ReadCloser implementation
