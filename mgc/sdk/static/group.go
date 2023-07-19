package static

import (
	"fmt"
	"sdk"
)

type StaticGroup struct {
	name        string
	version     string
	description string
	children    []sdk.Descriptor
}

func NewStaticGroup(name string, version string, description string, children []sdk.Descriptor) *StaticGroup {
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

func (o *StaticGroup) VisitChildren(visitor sdk.DescriptorVisitor) (finished bool, err error) {
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

func (o *StaticGroup) GetChildByName(name string) (child sdk.Descriptor, err error) {
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
		return nil, fmt.Errorf("Child not found: %s", name)
	}

	return found, err
}

var _ sdk.Grouper = (*StaticGroup)(nil)

// END: Grouper interface
