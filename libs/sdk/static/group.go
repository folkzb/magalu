package mgc_static

import (
	"fmt"
	"mgc_sdk"
)

type StaticGroup struct {
	name        string
	version     string
	description string
	children    []mgc_sdk.Descriptor
}

func NewStaticGroup(name string, version string, description string, children []mgc_sdk.Descriptor) *StaticGroup {
	return &StaticGroup{name, version, description, children}
}

// BEGIN: Descriptor interface:

func (o *StaticGroup) Name() string {
	return o.name
}

func (o *StaticGroup) Version() string {
	return o.version
}

func (o *StaticGroup) Description() string {
	return o.description
}

// END: Descriptor interface

// BEGIN: Grouper interface:

func (o *StaticGroup) VisitChildren(visitor mgc_sdk.DescriptorVisitor) (finished bool, err error) {
	for _, c := range o.children {
		run, err := visitor(c)
		if err != nil {
			return false, err
		}
		if !run {
			return false, nil
		}
	}

	return true, nil
}

func (o *StaticGroup) GetChildByName(name string) (child mgc_sdk.Descriptor, err error) {
	var found mgc_sdk.Descriptor
	finished, err := o.VisitChildren(func(child mgc_sdk.Descriptor) (run bool, err error) {
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
		return nil, fmt.Errorf("Child not found: %s", name)
	}

	return found, err
}

var _ mgc_sdk.Grouper = (*StaticGroup)(nil)

// END: Grouper interface
