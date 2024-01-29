package blueprint

import (
	"errors"
	"fmt"

	"magalu.cloud/core"
)

type childSpec struct {
	core.DescriptorSpec
	grouperSpec
	executorSpec
	Ref string `json:"$ref" yaml:"$ref"`
}

func (c *childSpec) validate() (err error) {
	isGrouper := !c.grouperSpec.isEmpty()
	isExecutor := !c.executorSpec.isEmpty()

	if isGrouper && isExecutor {
		return &core.ChainedError{
			Name: c.DescriptorSpec.Name,
			Err:  errors.New("cannot be both group and executor"),
		}
	}

	// If Ref is present, validation will occur in 'newChild'
	if c.Ref != "" {
		return nil
	}

	err = c.DescriptorSpec.Validate()
	if err != nil {
		return err
	}

	if isGrouper {
		err := c.grouperSpec.validate()
		if err != nil {
			return &core.ChainedError{
				Name: c.DescriptorSpec.Name,
				Err:  fmt.Errorf("invalid group definition: %w", err),
			}
		}
		return nil
	}

	if isExecutor {
		err := c.executorSpec.validate()
		if err != nil {
			return &core.ChainedError{
				Name: c.DescriptorSpec.Name,
				Err:  fmt.Errorf("invalid executor definition: %w", err),
			}
		}
		return nil
	}

	return &core.ChainedError{
		Name: c.DescriptorSpec.Name,
		Err:  errors.New("child must be either a group or an executor"),
	}
}
