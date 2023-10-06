package dataloader

import (
	"fmt"
	"os"
	"path"
)

type FileLoader struct {
	Dir string
}

func (f FileLoader) Load(name string) ([]byte, error) {
	return os.ReadFile(path.Join(f.Dir, name))
}

func (f FileLoader) String() string {
	return fmt.Sprintf("FileLoader(dir: %s)", f.Dir)
}

var _ Loader = (*FileLoader)(nil)
