package blueprint

import (
	"fmt"
	"os"

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

const indexFileName = "index.blueprint.yaml"
const indexVersion = "1.0.0"

// Source -> Module -> Group -> Executor

// -- ROOT: Source

const (
	mgcSdkDocumentUrl     = "http://magalu.cloud/sdk" // url to access Sdk.Group() (executor's root)
	currentUrlPlaceholder = "blueprint"               // replaced with the given CurrentUrl
)

func NewSource(loader dataloader.Loader, rootRefResolver core.RefPathResolver) core.Grouper {
	return core.NewSimpleGrouper(
		core.DescriptorSpec{
			Name:        "Blueprints",
			Description: fmt.Sprintf("Blueprints loaded using %v", loader),
		},
		func() (modules []core.Grouper, err error) {
			refResolver := core.NewMultiRefPathResolver()
			refResolver.EmptyDocumentUrl = mgcSdkDocumentUrl
			refResolver.CurrentUrlPlaceholder = currentUrlPlaceholder
			err = refResolver.Add(mgcSdkDocumentUrl, rootRefResolver)
			if err != nil {
				return
			}

			data, err := loader.Load(indexFileName)
			if err != nil {
				if os.IsNotExist(err) {
					// blueprint is not mandatory
					return []core.Grouper{}, nil
				}
				return
			}

			var index indexFileSpec
			err = yaml.Unmarshal(data, &index)
			if err != nil {
				return
			}
			if index.Version != indexVersion {
				return nil, fmt.Errorf("unsupported %q version %q, expected %q", indexFileName, index.Version, indexVersion)
			}

			modules = make([]core.Grouper, len(index.Modules))

			for i := range index.Modules {
				var module core.Grouper
				module, err = newModule(
					&index.Modules[i],
					loader,
					logger(),
					refResolver,
				)
				if err != nil {
					return
				}
				modules[i] = module
			}

			return
		},
	)
}
