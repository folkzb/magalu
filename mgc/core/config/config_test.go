package config

import (
	"path"
	"testing"

	"github.com/spf13/afero"
	"magalu.cloud/core"
)

type test struct {
	key      string
	fileData []byte
	expected any
}

func setupWithoutFile() *Config {
	path, _ := core.BuildMGCPath()
	c := New()
	c.init(path, afero.NewMemMapFs())

	return c
}

func setupWithFile(testFileData []byte) (*Config, error) {
	file, err := core.BuildMGCFilePath(CONFIG_FILE)
	if err != nil {
		return nil, err
	}

	fs := afero.NewMemMapFs()
	if err := afero.WriteFile(fs, file, testFileData, 0644); err != nil {
		return nil, err
	}

	c := New()
	c.init(path.Dir(file), fs)

	return c, nil
}

func TestGetWithoutFile(t *testing.T) {
	tests := []test{
		{key: "foo", fileData: []byte{}, expected: nil},
	}

	for _, tc := range tests {
		c := setupWithoutFile()

		if v := c.Get(tc.key); v != tc.expected {
			t.Errorf("expected %v, found %v", tc.expected, v)
		}
	}
}

func TestGetWithFile(t *testing.T) {
	tests := []test{
		{key: "foo", fileData: []byte(`foo: bar`), expected: "bar"},
		{key: "foo", fileData: []byte(`foo:`), expected: nil},
		{key: "foo", fileData: []byte(``), expected: nil},
	}

	for _, tc := range tests {
		c, err := setupWithFile(tc.fileData)
		if err != nil {
			t.Errorf("expected err == nil, found: %v", err)
		}

		if v := c.Get(tc.key); v != tc.expected {
			t.Errorf("expected %v, found %v", tc.expected, v)
		}
	}
}

func TestGetEnvVar(t *testing.T) {
	t.Setenv("MGC_FOO", "bar")
	c := setupWithoutFile()

	if v := c.Get("foo"); v != "bar" {
		t.Errorf("expected %v, found %v", "foo", v)
	}
}

func TestSetWithoutFile(t *testing.T) {
	tests := []test{
		{key: "foo", fileData: []byte{}, expected: "woo"},
	}

	for _, tc := range tests {
		c := setupWithoutFile()

		if err := c.Set(tc.key, tc.expected); err != nil {
			t.Errorf("expected err == nil , found %v", err)
		}

		if v := c.Get(tc.key); v != tc.expected {
			t.Errorf("expected %v, found %v", tc.expected, v)
		}
	}
}

func TestSetWithFile(t *testing.T) {
	tests := []test{
		{key: "foo", fileData: []byte("foo: bar"), expected: "woo"},
		{key: "foo", fileData: []byte("foo:"), expected: "woo"},
		{key: "foo", fileData: []byte(""), expected: "woo"},
	}

	for _, tc := range tests {
		c, err := setupWithFile(tc.fileData)
		if err != nil {
			t.Errorf("expected err == nil, found: %v", err)
		}

		if err := c.Set(tc.key, tc.expected); err != nil {
			t.Errorf("expected err == nil , found %v", err)
		}

		if v := c.Get(tc.key); v != tc.expected {
			t.Errorf("expected %v, found %v", tc.expected, v)
		}
	}
}

func TestDeleteWithoutFile(t *testing.T) {
	tests := []test{
		{key: "foo", fileData: []byte("foo: bar"), expected: nil},
		{key: "foo", fileData: []byte("foo:"), expected: nil},
		{key: "foo", fileData: []byte(""), expected: nil},
	}

	for _, tc := range tests {
		c := setupWithoutFile()

		if err := c.Delete(tc.key); err != nil {
			t.Errorf("expected err == nil, found %v", err)
		}

		if v := c.Get(tc.key); v != tc.expected {
			t.Errorf("expected %v, found %v", tc.expected, v)
		}
	}
}

func TestDeleteWithFile(t *testing.T) {
	tests := []test{
		{key: "foo", fileData: []byte("foo: bar"), expected: nil},
		{key: "foo", fileData: []byte("foo:"), expected: nil},
		{key: "foo", fileData: []byte(""), expected: nil},
	}

	for _, tc := range tests {
		c, err := setupWithFile(tc.fileData)
		if err != nil {
			t.Errorf("expected err == nil, found: %v", err)
		}

		if err := c.Delete(tc.key); err != nil {
			t.Errorf("expected err == nil, found %v", err)
		}

		if v := c.Get(tc.key); v != tc.expected {
			t.Errorf("expected %v, found %v", tc.expected, v)
		}
	}
}
