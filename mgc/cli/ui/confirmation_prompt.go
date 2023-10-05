package ui

import (
	"github.com/erikgeiser/promptkit/confirmation"
)

func Confirm(message string) (bool, error) {
	input := confirmation.New(message, confirmation.No)

	ready, err := input.RunPrompt()
	if err != nil {
		return false, err
	}

	return ready == *confirmation.Yes, nil
}
