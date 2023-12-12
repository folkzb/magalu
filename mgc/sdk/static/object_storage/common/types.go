package common

import (
	"fmt"
	"path"

	"github.com/invopop/jsonschema"

	mgcSchemaPkg "magalu.cloud/core/schema"
)

type BucketName string

func (b BucketName) JSONSchemaExtend(s *jsonschema.Schema) {
	// https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucketnamingrules.html
	s.MinLength = 3
	s.MaxLength = 63
}

func (b BucketName) AsURI(pathParts ...string) mgcSchemaPkg.URI {
	p := string(b)
	if len(pathParts) > 0 {
		parts := make([]string, 0, 1+len(pathParts))
		parts = append(parts, p)
		parts = append(parts, pathParts...)
		p = path.Join(parts...)
	}
	return mgcSchemaPkg.URI(p)
}

func (b BucketName) String() string {
	return string(b)
}

func NewBucketNameFromURI(u mgcSchemaPkg.URI) BucketName {
	return BucketName(path.Dir(u.Path()))
}

type ObjectError struct {
	Url mgcSchemaPkg.URI
	Err error
}

func (e *ObjectError) Error() string {
	return fmt.Sprintf("%s - %s, ", e.Url, e.Err.Error())
}

func (e *ObjectError) Unwrap() error {
	return e.Err
}
