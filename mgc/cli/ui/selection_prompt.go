package ui

import (
	"fmt"

	"github.com/erikgeiser/promptkit/selection"
)

func SelectionPrompt[C ~[]T, T any](msg string, choices C) (choice T, err error) {
	sel := selection.New(msg, choices)
	sel.LoopCursor = true

	return sel.RunPrompt()
}

type multipleSelectionDone string

const multipleSelectionDoneValue = multipleSelectionDone("⏎ Done")

type multipleSelectionChoice struct {
	value      any
	isSelected bool
}

func (c *multipleSelectionChoice) String() string {
	if c.value == multipleSelectionDoneValue {
		return string(multipleSelectionDoneValue)
	}

	var mark string
	if c.isSelected {
		mark = "✔"
	} else {
		mark = " "
	}
	return fmt.Sprintf("%s %s", mark, c.value)
}

// poor's man version since promptkit doesn't support it natively yet:
// https://github.com/erikgeiser/promptkit/issues/2
func MultiSelectionPrompt[C ~[]T, T any](msg string, choices C) (selected C, err error) {
	items := make([]*multipleSelectionChoice, 0, len(choices)+1)
	items = append(items, &multipleSelectionChoice{value: multipleSelectionDoneValue})
	for _, c := range choices {
		items = append(items, &multipleSelectionChoice{value: c})
	}

	// prints the prompt message to avoid repeating it when executing the multiple selection prompt
	fmt.Println(msg)

	for {
		var c *multipleSelectionChoice
		c, err = SelectionPrompt("", items)
		if err != nil {
			return
		}
		if c.value == multipleSelectionDoneValue {
			break
		}
		c.isSelected = !c.isSelected
	}

	for _, item := range items {
		if item.isSelected {
			selected = append(selected, item.value.(T))
		}
	}

	return
}
