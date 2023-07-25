package openapi

import (
	"context"
	"fmt"

	"magalu.cloud/core"

	"github.com/getkin/kin-openapi/openapi3"
)

// Source -> Module -> Resource -> Operation

// Module

type Module struct {
	name            string
	path            string
	extensionPrefix *string
	doc             *openapi3.T
}

// BEGIN: Descriptor interface:

func (m *Module) Name() string {
	return m.name
}

func (m *Module) Version() string {
	return "TODO: load version from index"
}

func (m *Module) Description() string {
	return "TODO: load description from index"
}

// END: Descriptor interface

// BEGIN: Grouper interface:

func (m *Module) getDoc() (*openapi3.T, error) {
	if m.doc == nil {
		ctx := context.Background()
		loader := openapi3.Loader{Context: ctx, IsExternalRefsAllowed: true}
		doc, err := loader.LoadFromFile(m.path)
		if err != nil {
			return nil, err
		}
		m.doc = doc
	}

	return m.doc, nil
}

func (m *Module) VisitChildren(visitor core.DescriptorVisitor) (finished bool, err error) {
	doc, err := m.getDoc()
	if err != nil {
		return false, err
	}

	for _, tag := range doc.Tags {
		if getHiddenExtension(m.extensionPrefix, tag.Extensions) {
			continue
		}

		resource := &Resource{
			tag:             tag,
			doc:             doc,
			extensionPrefix: m.extensionPrefix,
			servers:         doc.Servers,
		}

		run, err := visitor(resource)
		if err != nil {
			return false, err
		}
		if !run {
			return false, nil
		}
	}

	return true, nil
}

func (m *Module) GetChildByName(name string) (child core.Descriptor, err error) {
	// TODO: write O(1) version that doesn't list
	var found core.Descriptor
	finished, err := m.VisitChildren(func(child core.Descriptor) (run bool, err error) {
		if child.Name() == name {
			found = child
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	if finished {
		return nil, fmt.Errorf("Resource not found: %s", name)
	}

	return found, err
}

var _ core.Grouper = (*Module)(nil)

// END: Grouper interface