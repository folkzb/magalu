package ui

import (
	"fmt"

	"github.com/erikgeiser/promptkit/selection"
)

func SelectionPromptChoice(msg string, choices []*SelectionChoice) (choice *SelectionChoice, err error) {
	sel := selection.New(msg, choices)
	sel.LoopCursor = true

	return sel.RunPrompt()
}

func SelectionPrompt[T any](msg string, choices []*SelectionChoice) (value T, err error) {
	choice, err := SelectionPromptChoice(msg, choices)
	if err != nil {
		return
	}
	value, ok := choice.Value.(T)
	if !ok {
		err = fmt.Errorf("expected type %T, got %T (%#v)", *new(T), choice.Value, choice.Value)
		return
	}
	return
}

type multipleSelectionDone string

const multipleSelectionDoneValue = multipleSelectionDone("⏎ Done")

type SelectionChoice struct {
	Value      any
	Label      string
	IsSelected bool
}

func (c *SelectionChoice) String() string {
	if c.Value == multipleSelectionDoneValue {
		return string(multipleSelectionDoneValue)
	}

	var mark, label string
	if c.IsSelected {
		mark = "✔"
	} else {
		mark = " "
	}
	if c.Label != "" {
		label = c.Label
	} else {
		label = fmt.Sprint(c.Value)
	}
	return fmt.Sprintf("%s %s", mark, label)
}

// poor's man version since promptkit doesn't support it natively yet:
// https://github.com/erikgeiser/promptkit/issues/2
func MultiSelectionPrompt[T any](msg string, choices []*SelectionChoice) (selected []T, err error) {
	items := make([]*SelectionChoice, 0, len(choices)+1)
	items = append(items, &SelectionChoice{Value: multipleSelectionDoneValue})
	items = append(items, choices...)

	// prints the prompt message to avoid repeating it when executing the multiple selection prompt
	fmt.Println(msg)

	for {
		var c *SelectionChoice
		c, err = SelectionPromptChoice("", items)
		if err != nil {
			return
		}
		if c.Value == multipleSelectionDoneValue {
			break
		}
		c.IsSelected = !c.IsSelected
	}

	for i, item := range items {
		if item.IsSelected {
			value, ok := item.Value.(T)
			if !ok {
				err = fmt.Errorf("item #%d expected type %T, got %T (%#v)", i, *new(T), item.Value, item.Value)
				return
			}
			selected = append(selected, value)
		}
	}

	return
}
