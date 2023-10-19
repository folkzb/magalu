package blueprint

import (
	"fmt"

	"go.uber.org/zap"
	"magalu.cloud/core"
	"magalu.cloud/core/dataloader"
	"magalu.cloud/core/utils"
)

// Module -> Group -> Executor

// Module

func newDocumentLoader(
	indexModule *indexModuleSpec,
	loader dataloader.Loader,
	logger *zap.SugaredLogger,
) utils.LoadWithError[*document] {
	return utils.NewLazyLoaderWithError(func() (doc *document, err error) {
		mData, err := loader.Load(indexModule.Path)
		if err != nil {
			return
		}

		doc, err = newDocumentFromData(mData)
		if err != nil {
			logger.Warnw(
				"unable to load blueprint module",
				"module", indexModule,
				"error", err,
				"data", string(mData),
			)
			return nil, fmt.Errorf("unable to load %q: %w", indexModule.Path, err)
		}

		err = doc.validate()
		if err != nil {
			logger.Warnw(
				"invalid blueprint module",
				"module", indexModule,
				"error", err,
				"doc", doc,
			)
			return nil, fmt.Errorf("invalid blueprint module %q: %w", indexModule.Path, err)
		}

		return
	})
}

func newModule(
	indexModule *indexModuleSpec,
	loader dataloader.Loader,
	logger *zap.SugaredLogger,
	refResolver *core.MultiRefPathResolver,
) (m core.Grouper, err error) {
	logger = logger.Named(indexModule.Name)
	loadDoc := newDocumentLoader(indexModule, loader, logger)

	docResolver := core.NewDocumentRefPathResolver(func() (any, error) { return loadDoc() })
	err = refResolver.Add(indexModule.Url, docResolver)
	if err != nil {
		return
	}

	return core.NewSimpleGrouper(
		indexModule.DescriptorSpec,
		func() (children []core.Descriptor, err error) {
			doc, err := loadDoc()
			if err != nil {
				return
			}

			boundRefResolver := core.NewBoundRefResolver(indexModule.Url, refResolver)
			return createGroupChildren(doc.Children, logger, boundRefResolver)
		},
	), nil
}
