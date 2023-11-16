package core

type UsageError struct {
	Err error
}

func (e UsageError) Unwrap() error {
	return e.Err
}

func (e UsageError) Error() string {
	return e.Err.Error()
}
