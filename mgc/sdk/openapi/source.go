package openapi

import (
	"fmt"

	"gopkg.in/yaml.v3"
	"magalu.cloud/core"
	"magalu.cloud/core/dataloader"
)

type indexModuleSpec struct {
	Name        string
	Url         string
	Path        string
	Version     string
	Description string
}

type indexFileSpec struct {
	Version string
	Modules []indexModuleSpec
}

const indexFileName = "index.openapi.yaml"
const indexVersion = "1.0.0"

// Source -> Module -> Resource -> Operation

// -- ROOT: Source

type source struct {
	Loader dataloader.Loader
	*core.GrouperLazyChildren[*module]
}

// BEGIN: Descriptor interface:

func (o *source) Name() string {
	return "OpenApis"
}

func (o *source) Version() string {
	return ""
}

func (o *source) Description() string {
	return fmt.Sprintf("OpenApis loaded using %v", o.Loader)
}

// END: Descriptor interface

// BEGIN: Grouper interface:

func NewSource(loader dataloader.Loader, extensionPrefix *string) *source {
	return &source{
		Loader: loader,
		GrouperLazyChildren: core.NewGrouperLazyChildren[*module](
			func() (modules []*module, err error) {
				data, err := loader.Load(indexFileName)
				if err != nil {
					return nil, err
				}

				var index indexFileSpec
				err = yaml.Unmarshal(data, &index)
				if err != nil {
					return nil, err
				}
				if index.Version != indexVersion {
					return nil, fmt.Errorf("unsupported %q version %q, expected %q", indexFileName, index.Version, indexVersion)
				}

				modules = make([]*module, len(index.Modules))
				moduleResolver := moduleResolver{}

				for i, item := range index.Modules {
					module := newModule(
						item,
						extensionPrefix,
						loader,
						logger(),
					)
					modules[i] = module
					moduleResolver.add(module)
				}

				return modules, nil
			}),
	}
}

// implemented by embedded GrouperLazyChildren
var _ core.Grouper = (*source)(nil)

// END: Grouper interface
