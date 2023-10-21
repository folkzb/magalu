package openapi

import (
	"context"

	"go.uber.org/zap"
	"magalu.cloud/core"
	"magalu.cloud/core/dataloader"

	"github.com/getkin/kin-openapi/openapi3"
)

// Source -> Module -> Resource -> Operation

// Module

type module struct {
	core.SimpleDescriptor
	execResolver executorResolver
	loaded       bool
	*core.GrouperLazyChildren[*resource]
}

func newModule(
	indexModule indexModuleSpec,
	extensionPrefix *string,
	loader dataloader.Loader,
	logger *zap.SugaredLogger,
) (m *module) {
	logger = logger.Named(indexModule.Name)
	m = &module{
		SimpleDescriptor: core.SimpleDescriptor{Spec: indexModule.DescriptorSpec},
		GrouperLazyChildren: core.NewGrouperLazyChildren[*resource](func() (resources []*resource, err error) {
			ctx := context.Background()
			mData, err := loader.Load(indexModule.Path)
			if err != nil {
				return nil, err
			}

			oapiLoader := openapi3.Loader{Context: ctx, IsExternalRefsAllowed: false}
			doc, err := oapiLoader.LoadFromData(mData)
			if err != nil {
				return nil, err
			}

			resources = make([]*resource, 0, len(doc.Tags))

			for _, tag := range doc.Tags {
				if getHiddenExtension(extensionPrefix, tag.Extensions) {
					continue
				}

				resource := newResource(
					tag,
					doc,
					extensionPrefix,
					logger,
					m,
				)

				resources = append(resources, resource)
			}

			return resources, nil
		}),
	}
	return m
}

func (m *module) loadRecursive() {
	if m.loaded {
		return
	}
	// Recursively load the whole module to guarantee resolverTree is known
	var loadRecursive func(child core.Descriptor) (run bool, err error)
	loadRecursive = func(child core.Descriptor) (run bool, err error) {
		if group, ok := child.(core.Grouper); ok {
			return group.VisitChildren(loadRecursive)
		}
		return true, nil
	}
	_, _ = m.VisitChildren(loadRecursive)
	m.loaded = true
}

// implemented by embedded GrouperLazyChildren & SimpleDescriptor
var _ core.Grouper = (*module)(nil)
