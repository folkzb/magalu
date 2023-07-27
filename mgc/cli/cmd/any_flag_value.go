package cmd

type AnyFlagValue struct {
	marshalledValue string
	typeName        string
}

func (f *AnyFlagValue) String() string {
	return f.marshalledValue
}

func (f *AnyFlagValue) Set(val string) error {
	f.marshalledValue = val
	return nil
}

func (f *AnyFlagValue) Type() string {
	return f.typeName
}
