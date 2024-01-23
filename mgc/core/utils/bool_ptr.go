package utils

func BoolPtr(b bool) *bool {
	boolVar := b
	return &boolVar
}
