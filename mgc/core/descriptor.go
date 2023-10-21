package core

import (
	"errors"
	"fmt"
)

type DescriptorSpec struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

func (d *DescriptorSpec) Validate() error {
	if d.Name == "" {
		return &ChainedError{fmt.Sprintf("<missing name %p>", d), errors.New("missing name")}
	}
	if d.Description == "" {
		return &ChainedError{d.Name, errors.New("missing description")}
	}
	// Version is optional
	return nil
}

// General interface that describes both Executor and Grouper
type Descriptor interface {
	Name() string
	Version() string
	Description() string
	DescriptorSpec() DescriptorSpec
}

type SimpleDescriptor struct {
	Spec DescriptorSpec
}

func (d *SimpleDescriptor) Name() string {
	return d.Spec.Name
}

func (d *SimpleDescriptor) Version() string {
	return d.Spec.Version
}

func (d *SimpleDescriptor) Description() string {
	return d.Spec.Description
}

func (d *SimpleDescriptor) DescriptorSpec() DescriptorSpec {
	return d.Spec
}

var _ Descriptor = (*SimpleDescriptor)(nil)

type DescriptorVisitor func(child Descriptor) (run bool, err error)
