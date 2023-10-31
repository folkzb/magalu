package cmd

import (
	"encoding/json"
	"os"
	"strings"
)

type anyFlagValue struct {
	value    any
	typeName string
}

func (f *anyFlagValue) String() string {
	if f.value == nil {
		return ""
	}

	str, err := json.Marshal(f.value)
	if err != nil {
		return ""
	}
	return string(str)
}

func (f *anyFlagValue) Set(val string) error {
	var err error
	switch {
	case strings.HasPrefix(val, "@"):
		f.value, err = loadJSONFromFile(val[1:])
		if err != nil {
			return err
		}
	case strings.HasPrefix(val, "%"):
		f.value, err = loadFromFile(val[1:])
		if err != nil {
			return err
		}
	case strings.HasPrefix(val, "#"):
		f.value = val[1:]
	default:
		err := json.Unmarshal([]byte(val), &f.value)
		if err != nil {
			f.value = val
		}
	}
	return nil
}

func (f *anyFlagValue) Type() string {
	return f.typeName
}

func (f *anyFlagValue) Value() any {
	return f.value
}

func loadJSONFromFile(filename string) (any, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var value any
	err = json.Unmarshal(data, &value)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func loadFromFile(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
