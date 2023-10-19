//go:build release

package sdk

import _ "embed"

//go:embed version.txt
var Version string
