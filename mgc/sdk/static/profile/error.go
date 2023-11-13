package profile

import "fmt"

type ProfileError struct {
	Name string
	Err  error
}

func (e ProfileError) Unwrap() error {
	return e.Err
}

func (e ProfileError) Error() string {
	return fmt.Sprintf("profle %s: %s", e.Name, e.Err.Error())
}
