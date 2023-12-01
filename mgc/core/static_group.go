package core

import (
	"fmt"
	"slices"
	"strings"
)

type StaticGroup struct {
	SimpleDescriptor
	children []Descriptor
}

func NewStaticGroup(spec DescriptorSpec, children []Descriptor) *StaticGroup {
	slices.SortFunc(children, func(a, b Descriptor) int {
		return strings.Compare(a.Name(), b.Name())
	})

	return &StaticGroup{SimpleDescriptor{spec}, children}
}

func (o *StaticGroup) VisitChildren(visitor DescriptorVisitor) (finished bool, err error) {
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

func (o *StaticGroup) GetChildByName(name string) (child Descriptor, err error) {
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
		return nil, fmt.Errorf("child not found: %s", name)
	}

	return found, err
}

// implemented by embedded SimpleDescriptor
var _ Grouper = (*StaticGroup)(nil)
