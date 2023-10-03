package cmd

import (
	"os"
	"strconv"

	"golang.org/x/term"
)

const (
	defaultTerminalColumns = 160
	defaultTerminalRows    = 160
)

// No env vars are checked, it's the reporter TTY value or defaults
func getTermSize() (columns int, rows int) {
	columns = defaultTerminalColumns
	rows = defaultTerminalRows
	if !term.IsTerminal(0) {
		return
	}

	columns, rows, _ = term.GetSize(0)
	if columns < 1 {
		columns = defaultTerminalColumns
	}
	if rows < 1 {
		rows = defaultTerminalRows
	}
	return
}

func getTermColumns() int {
	env := os.Getenv("COLUMNS")
	if termColumns, err := strconv.Atoi(env); err == nil && termColumns > 0 {
		return termColumns
	}
	columns, _ := getTermSize()
	return columns
}
