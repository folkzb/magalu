package openapi

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"sdk"
)

// Source -> Module -> Resource -> Operation

// -- ROOT: Source

type Source struct {
	Dir             string
	ExtensionPrefix *string
}

// BEGIN: Descriptor interface:

func (o *Source) Name() string {
	return "OpenApis"
}

func (o *Source) Version() string {
	return ""
}

func (o *Source) Description() string {
	return fmt.Sprintf("OpenApis loaded from %v", o.Dir)
}

// END: Descriptor interface

// BEGIN: Grouper interface:

var openAPIFileNameRe = regexp.MustCompile("^(?P<name>[^.]+)(?:|[.]openapi)[.](?P<ext>json|yaml|yml)$")

func (o *Source) VisitChildren(visitor sdk.DescriptorVisitor) (finished bool, err error) {
	// TODO: load from an index with description + version information

	items, err := os.ReadDir(o.Dir)
	if err != nil {
		return false, fmt.Errorf("Unable to read OpenAPI files at %s: %w", o.Dir, err)
	}

	for _, item := range items {
		info, err := item.Info()
		if err != nil {
			continue
		}

		if info.IsDir() {
			continue
		}

		matches := openAPIFileNameRe.FindStringSubmatch(item.Name())

		if len(matches) == 0 {
			continue
		}

		module := &Module{
			name:            matches[1],
			path:            filepath.Join(o.Dir, item.Name()),
			extensionPrefix: o.ExtensionPrefix,
		}

		run, err := visitor(module)
		if err != nil {
			return false, err
		}
		if !run {
			return false, nil
		}
	}

	return true, nil
}

func (o *Source) GetChildByName(name string) (child sdk.Descriptor, err error) {
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
		return nil, fmt.Errorf("Module not found: %s", name)
	}

	return found, err
}

var _ sdk.Grouper = (*Source)(nil)

// END: Grouper interface
