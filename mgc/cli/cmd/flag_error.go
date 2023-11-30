package cmd

import (
	"fmt"

	flag "github.com/spf13/pflag"
)

type flagError struct {
	Flag *flag.Flag
	Err  error
}

var _ error = (*flagError)(nil)

func (e *flagError) Error() string {
	return fmt.Sprintf("flag \"--%s=%s\" error: %s", e.Flag.Name, e.Flag.Value.String(), e.Err.Error())
}

func (e *flagError) Unwrap() error {
	return e.Err
}
