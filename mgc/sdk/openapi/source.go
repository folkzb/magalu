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

func NewSource(loader dataloader.Loader, extensionPrefix *string) *core.SimpleGrouper[core.Grouper] {
	return core.NewSimpleGrouper(
		core.DescriptorSpec{
			Name:        "OpenApis",
			Description: fmt.Sprintf("OpenApis loaded using %v", loader),
		},
		func() (modules []core.Grouper, err error) {
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

			modules = make([]core.Grouper, len(index.Modules))
			refResolver := core.NewMultiRefPathResolver()
			for i := range index.Modules {
				var module core.Grouper
				module, err = newModule(
					&index.Modules[i],
					extensionPrefix,
					loader,
					logger(),
					refResolver,
				)
				if err != nil {
					return
				}
				modules[i] = module
			}

			return modules, nil
		},
	)
}
