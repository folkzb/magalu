package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/PaesslerAG/jsonpath"
)

type jsonpathFileOutputFormatter struct{}

func (*jsonpathFileOutputFormatter) Format(value any, options string) error {
	filename := options
	path, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("jsonpath-file output formatter: %w", err)
	}
	target, err := jsonpath.Get(string(path), value)
	if err != nil {
		return fmt.Errorf("jsonpath-file output formatter: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", " ")
	return enc.Encode(target)
}

func init() {
	outputFormatters["jsonpath-file"] = &jsonpathFileOutputFormatter{}
}
