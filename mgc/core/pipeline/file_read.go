package pipeline

import (
	"context"
	"fmt"
	"io"
	"os"
)

// ReadChunks -> ParallelProcessChunks -> processor

type ChunkReader interface {
	io.Reader
	io.ReaderAt
}

type Chunk struct {
	Reader      ChunkReader
	StartOffset int64
	TotalSize   int64
}

// Reads a file into Chunks (Sections), each being its own ChunkReader (io.SectionReader)
//
// Wrapper on top of ReadChunks() that queries the f.Stat().Size()
func ReadFileChunks(ctx context.Context, f *os.File, chunkSize int64) (<-chan Chunk, error) {
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return ReadChunks(ctx, f, st.Size(), chunkSize), nil
}

// Reads content splitting into Chunks (Sections), each being its own ChunkReader (io.SectionReader)
//
// Chunks are produced in a channel that is closed when the last is produced.
// The channel is not buffered, it won't send until the other side is ready to consume.
//
// Generation may be early stopped by context.Context.Done(), see
// context.WithCancel(), context.WithTimeout() and context.WithDeadline()
func ReadChunks(
	ctx context.Context,
	r io.ReaderAt,
	size int64,
	chunkSize int64,
) (outputChan <-chan Chunk) {
	ch := make(chan Chunk)
	outputChan = ch

	logger := FromContext(ctx).Named("ReadChunks").With(
		"readerAt", r,
		"size", size,
		"chunkSize", chunkSize,
		"outputChan", fmt.Sprintf("%#v", outputChan),
	)
	ctx = NewContext(ctx, logger)

	generator := func() {
		defer func() {
			logger.Info("closing output channel")
			close(ch)
		}()

		var i int64
		for i = 0; i < size; i += chunkSize {
			select {
			case <-ctx.Done():
				logger.Debugw("context.Done()", "err", ctx.Err())
				return

			case ch <- Chunk{io.NewSectionReader(r, i, chunkSize), i, size}:
				logger.Debugw("read section", "offset", i)
			}
		}
		logger.Debug("finished reading sections")
	}

	logger.Info("start")
	go generator()
	return
}
