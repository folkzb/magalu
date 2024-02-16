package ui

import (
	"github.com/erikgeiser/promptkit/selection"
)

func SelectionPrompt[C ~[]T, T any](msg string, choices C) (choice T, err error) {
	sel := selection.New(msg, choices)
	sel.LoopCursor = true

	return sel.RunPrompt()
}
