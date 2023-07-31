package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/PaesslerAG/jsonpath"
)

type jsonpathOutputFormatter struct{}

func (*jsonpathOutputFormatter) Format(value any, options string) error {
	path := options
	target, err := jsonpath.Get(path, value)
	if err != nil {
		return fmt.Errorf("jsonpath output formatter: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", " ")
	return enc.Encode(target)
}

func init() {
	outputFormatters["jsonpath"] = &jsonpathOutputFormatter{}
}
