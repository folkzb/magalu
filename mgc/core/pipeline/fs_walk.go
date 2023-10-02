package pipeline

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// WalkDirEntries -> FilterWalkDirEntries
// See fs.WalkDirFunc documentation
type WalkDirEntry struct {
	Path     string
	DirEntry fs.DirEntry
	Err      error
}

// Do not process any entry that name stars with "."
func WalkDirFilterHidden(path string, d fs.DirEntry, err error) error {
	if d == nil {
		return nil
	}
	if name := d.Name(); strings.HasPrefix(name, ".") {
		return fs.SkipDir
	}
	return nil
}

// Do not process any directory/folder that name stars with "."
func WalkDirFilterHiddenFolders(path string, d fs.DirEntry, err error) error {
	if d != nil && d.IsDir() {
		return WalkDirFilterHidden(path, d, err)
	}
	return nil
}

// WalkDirEntries recursively scans files/directories from a root directory
//
// checkPath() may be used to return fs.SkipDir or fs.SkipAll and control the walk process.
// If provided (non-nil), it's called before anything else. See fs.WalkDirFunc documentation.
// It may be used to omit hidden folders (ie: ".git") and the likes
//
// Each file/directory may contain an associated error, it may be ignored or keep going.
// By default, if no checkPath is provided, it keeps going.
func WalkDirEntries(
	ctx context.Context,
	root string,
	checkPath fs.WalkDirFunc,
) (outputChan <-chan WalkDirEntry) {
	ch := make(chan WalkDirEntry)
	outputChan = ch

	logger := FromContext(ctx).Named("WalkDirEntries").With(
		"root", root,
		"outputChan", fmt.Sprintf("%#v", outputChan),
	)
	ctx = NewContext(ctx, logger)

	generator := func() {
		defer func() {
			logger.Info("closing output channel")
			close(ch)
		}()

		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if d == nil {
				return filepath.SkipDir
			}
			if checkPath != nil {
				e := checkPath(path, d, err)
				if e != nil {
					logger.Debugw("checkPath != nil", "err", err, "path", path, "dirEntry", d)
					return e
				}
			}
			select {
			case <-ctx.Done():
				logger.Debugw("context.Done()", "err", ctx.Err())
				return filepath.SkipAll

			case ch <- WalkDirEntry{path, d, err}:
				logger.Debugw("entry", "err", err, "path", path, "dirEntry", d)
				return nil
			}
		})
		logger.Debug("finished walking entries")
	}

	logger.Info("start", root)
	go generator()
	return
}
