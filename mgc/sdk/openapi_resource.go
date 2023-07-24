package sdk

import (
	"fmt"

	"golang.org/x/exp/slices"

	"github.com/getkin/kin-openapi/openapi3"
)

// Source -> Module -> Resource -> Operation

// Resource

type OpenApiResource struct {
	tag             *openapi3.Tag
	doc             *openapi3.T
	extensionPrefix *string
}

// BEGIN: Descriptor interface:

func (o *OpenApiResource) Name() string {
	return getNameExtension(o.extensionPrefix, o.tag.Extensions, o.tag.Name)
}

func (o *OpenApiResource) Version() string {
	return ""
}

func (o *OpenApiResource) Description() string {
	return o.tag.Description
}

// END: Descriptor interface

// BEGIN: Grouper interface:

func (o *OpenApiResource) visitPath(key string, p *openapi3.PathItem, visitor DescriptorVisitor) (run bool, err error) {
	ops := map[string]*openapi3.Operation{
		"get":    p.Get,
		"post":   p.Post,
		"put":    p.Put,
		"patch":  p.Patch,
		"delete": p.Delete,
	}

	for method, op := range ops {
		if op == nil || getHiddenExtension(o.extensionPrefix, op.Extensions) {
			continue
		}

		if !slices.Contains(op.Tags, o.tag.Name) {
			continue
		}

		operation := &OpenApiOperation{
			key:             key,
			method:          method,
			path:            p,
			operation:       op,
			doc:             o.doc,
			extensionPrefix: o.extensionPrefix,
		}

		run, err := visitor(operation)
		if !run || err != nil {
			return false, err
		}
	}

	return true, nil
}

func (o *OpenApiResource) VisitChildren(visitor DescriptorVisitor) (finished bool, err error) {
	// TODO: provide optimized lookup
	for k, p := range o.doc.Paths {
		if getHiddenExtension(o.extensionPrefix, p.Extensions) {
			continue
		}

		run, err := o.visitPath(k, p, visitor)
		if err != nil {
			return false, err
		}
		if !run {
			return false, nil
		}
	}

	return true, nil
}

func (o *OpenApiResource) GetChildByName(name string) (child Descriptor, err error) {
	// TODO: write O(1) version that doesn't list
	var found Descriptor
	finished, err := o.VisitChildren(func(child Descriptor) (run bool, err error) {
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
		return nil, fmt.Errorf("Action not found: %s", name)
	}

	return found, err
}

var _ Grouper = (*OpenApiResource)(nil)

// END: Grouper interface
