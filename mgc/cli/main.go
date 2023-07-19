package main

import (
	"cli/cmd"
	"fmt"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		fmt.Printf("Error running the CLI: %s\n", err)
	}
}
