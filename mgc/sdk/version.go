package sdk

import (
	_ "embed"
	"os"
	"strings"
)

//go:embed version.txt
var rawVersion string

var version string = func() string {
	if vv := os.Getenv("VERSION"); vv != "" {
		if vv, ok := strings.CutPrefix(vv, "v"); ok {
			return vv
		}
		return vv
	}
	return strings.Trim(rawVersion, " \t\n\r")
}()
