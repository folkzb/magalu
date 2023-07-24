package sdk

import (
	"context"
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

// Source -> Module -> Resource -> Operation

// Module

type OpenApiModule struct {
	name            string
	path            string
	extensionPrefix *string
	doc             *openapi3.T
}

// BEGIN: Descriptor interface:

func (m *OpenApiModule) Name() string {
	return m.name
}

func (m *OpenApiModule) Version() string {
	return "TODO: load version from index"
}

func (m *OpenApiModule) Description() string {
	return "TODO: load description from index"
}

// END: Descriptor interface

// BEGIN: Grouper interface:

func (m *OpenApiModule) getDoc() (*openapi3.T, error) {
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

func (m *OpenApiModule) VisitChildren(visitor DescriptorVisitor) (finished bool, err error) {
	doc, err := m.getDoc()
	if err != nil {
		return false, err
	}

	for _, tag := range doc.Tags {
		if getHiddenExtension(m.extensionPrefix, tag.Extensions) {
			continue
		}

		resource := &OpenApiResource{
			tag:             tag,
			doc:             doc,
			extensionPrefix: m.extensionPrefix,
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

func (m *OpenApiModule) GetChildByName(name string) (child Descriptor, err error) {
	// TODO: write O(1) version that doesn't list
	var found Descriptor
	finished, err := m.VisitChildren(func(child Descriptor) (run bool, err error) {
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

var _ Grouper = (*OpenApiModule)(nil)

// END: Grouper interface
