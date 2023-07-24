package main

import (
	"fmt"

	"magalu.cloud/cli/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		fmt.Printf("Error running the CLI: %s\n", err)
	}
}
