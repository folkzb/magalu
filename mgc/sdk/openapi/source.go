package openapi

import (
	"fmt"

	"github.com/invopop/yaml"
	"magalu.cloud/core"
	"magalu.cloud/core/dataloader"
)

type indexModuleSpec struct {
	core.DescriptorSpec
	Url  string
	Path string
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
	core.SimpleDescriptor
	*core.GrouperLazyChildren[*module]
}

func NewSource(loader dataloader.Loader, extensionPrefix *string) *source {
	return &source{
		SimpleDescriptor: core.SimpleDescriptor{Spec: core.DescriptorSpec{
			Name:        "OpenApis",
			Description: fmt.Sprintf("OpenApis loaded using %v", loader),
		}},
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
					moduleResolver.add(item.Url, module)
				}

				return modules, nil
			}),
	}
}

// implemented by embedded GrouperLazyChildren & SimpleDescriptor
var _ core.Grouper = (*source)(nil)
