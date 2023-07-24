package openapi

import (
	"fmt"

	"golang.org/x/exp/slices"
	"magalu.cloud/core"

	"github.com/getkin/kin-openapi/openapi3"
)

// Source -> Module -> Resource -> Operation

// Resource

type Resource struct {
	tag             *openapi3.Tag
	doc             *openapi3.T
	extensionPrefix *string
	servers         openapi3.Servers
}

// BEGIN: Descriptor interface:

func (o *Resource) Name() string {
	return getNameExtension(o.extensionPrefix, o.tag.Extensions, o.tag.Name)
}

func (o *Resource) Version() string {
	return ""
}

func (o *Resource) Description() string {
	return o.tag.Description
}

// END: Descriptor interface

// BEGIN: Grouper interface:

func getServers(p *openapi3.PathItem, op *openapi3.Operation) openapi3.Servers {
	var servers openapi3.Servers
	if op.Servers != nil && len(*op.Servers) > 0 {
		servers = *op.Servers
	}
	if servers == nil && len(p.Servers) > 0 {
		servers = p.Servers
	}

	return servers
}

func (o *Resource) visitPath(key string, p *openapi3.PathItem, visitor core.DescriptorVisitor) (run bool, err error) {
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

		servers := getServers(p, op)
		if servers == nil {
			servers = o.servers
		}

		operation := &Operation{
			key:             key,
			method:          method,
			path:            p,
			operation:       op,
			doc:             o.doc,
			extensionPrefix: o.extensionPrefix,
			servers:         servers,
		}

		run, err := visitor(operation)
		if !run || err != nil {
			return false, err
		}
	}

	return true, nil
}

func (o *Resource) VisitChildren(visitor core.DescriptorVisitor) (finished bool, err error) {
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

func (o *Resource) GetChildByName(name string) (child core.Descriptor, err error) {
	// TODO: write O(1) version that doesn't list
	var found core.Descriptor
	finished, err := o.VisitChildren(func(child core.Descriptor) (run bool, err error) {
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

var _ core.Grouper = (*Resource)(nil)

// END: Grouper interface
