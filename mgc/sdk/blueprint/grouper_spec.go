package blueprint

import (
	"errors"
	"fmt"
)

type grouperSpec struct {
	Children []*childSpec `json:"children"`
}

func (g *grouperSpec) isEmpty() bool {
	return len(g.Children) == 0
}

func (g *grouperSpec) validate() error {
	if g.isEmpty() {
		return errors.New("no children")
	}

	for i, c := range g.Children {
		err := c.validate()
		if err != nil {
			return fmt.Errorf("invalid child %d: %w", i, err)
		}
	}

	return nil
}
