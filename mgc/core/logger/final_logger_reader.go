package logger

import (
	"bytes"
	"io"
)

type finalLoggerReader struct {
	parent      io.Reader
	buffer      bytes.Buffer
	lastReadErr error
	logger      func(readData LogReadData)
}

// Wraps an io.Reader in another Reader which accumulates the read data as the parent is read,
// then calls the logger at the end with the whole data.
// Errors will be logged as they occur.
func NewFinalLoggerReader(parent io.Reader, logger func(readData LogReadData)) io.ReadCloser {
	return &finalLoggerReader{
		parent: parent,
		logger: logger,
	}
}

func (pr *finalLoggerReader) Unwrap() io.Reader {
	return pr.parent
}

// BEGIN io.ReadCloser implementation

func (pr *finalLoggerReader) Read(p []byte) (n int, err error) {
	n, err = pr.parent.Read(p)
	if err != nil && err != io.EOF {
		pr.lastReadErr = err
		pr.logger(LogReadData{Size: n, Data: string(p[:n]), Err: err})
	}
	_, _ = pr.buffer.Write(p[:n])
	return
}

func (pr *finalLoggerReader) Close() error {
	pr.logger(LogReadData{Size: pr.buffer.Len(), Data: pr.buffer.String(), Err: pr.lastReadErr})
	if closer, ok := pr.parent.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

var _ io.ReadCloser = (*finalLoggerReader)(nil)

// END io.ReadCloser implementation
