package dataloader

// Given a name (path), return its contents
type Loader interface {
	Load(name string) ([]byte, error)
}
