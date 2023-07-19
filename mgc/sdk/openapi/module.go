package openapi

import (
	"context"
	"fmt"
	"sdk"

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

func (o *Module) Name() string {
	return o.name
}

func (o *Module) Version() string {
	return "TODO: load version from index"
}

func (o *Module) Description() string {
	return "TODO: load description from index"
}

// END: Descriptor interface

// BEGIN: Grouper interface:

func (o *Module) getDoc() (*openapi3.T, error) {
	if o.doc == nil {
		ctx := context.Background()
		loader := openapi3.Loader{Context: ctx, IsExternalRefsAllowed: true}
		doc, err := loader.LoadFromFile(o.path)
		if err != nil {
			return nil, err
		}
		o.doc = doc
	}

	return o.doc, nil
}

func (o *Module) VisitChildren(visitor sdk.DescriptorVisitor) (finished bool, err error) {
	doc, err := o.getDoc()
	if err != nil {
		return false, err
	}

	for _, tag := range doc.Tags {
		if getHiddenExtension(o.extensionPrefix, tag.Extensions) {
			continue
		}

		resource := &Resource{
			tag:             tag,
			doc:             doc,
			extensionPrefix: o.extensionPrefix,
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

func (o *Module) GetChildByName(name string) (child sdk.Descriptor, err error) {
	// TODO: write O(1) version that doesn't list
	var found sdk.Descriptor
	finished, err := o.VisitChildren(func(child sdk.Descriptor) (run bool, err error) {
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

var _ sdk.Grouper = (*Module)(nil)

// END: Grouper interface
