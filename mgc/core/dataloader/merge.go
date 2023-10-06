package dataloader

import (
	"fmt"
	"os"
	"syscall"
)

type MergeLoader struct {
	Loaders []Loader
}

func NewMergeLoader(loaders ...Loader) Loader {
	if len(loaders) == 1 {
		return loaders[0]
	}
	return &MergeLoader{loaders}
}

func (m *MergeLoader) Load(name string) (data []byte, err error) {
	for _, loader := range m.Loaders {
		data, err = loader.Load(name)
		if err == nil {
			return
		}
	}
	if err == nil {
		err = &os.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
	}
	return nil, err
}

func (m *MergeLoader) String() string {
	return fmt.Sprintf("MergeLoader(loaders: %s)", m.Loaders)
}

var _ Loader = (*MergeLoader)(nil)
