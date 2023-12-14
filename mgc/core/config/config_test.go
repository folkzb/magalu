package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"magalu.cloud/core/profile_manager"
)

type test struct {
	key      string
	fileData []byte
	expected any
}

func setupWithoutFile() *Config {
	pf, _ := profile_manager.NewInMemoryProfileManager()
	c := New(pf)
	c.init()

	return c
}

func setupWithFile(testFileData []byte) (*Config, error) {
	m, _ := profile_manager.NewInMemoryProfileManager()
	if err := m.Current().Write(CONFIG_FILE, testFileData); err != nil {
		return nil, err
	}

	c := New(m)

	return c, nil
}

func TestGetWithoutFile(t *testing.T) {
	tests := []test{
		{key: "foo", fileData: []byte{}, expected: nil},
	}

	for _, tc := range tests {
		c := setupWithoutFile()

		var out any
		if err := c.Get(tc.key, &out); err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}

		if out != tc.expected {
			t.Errorf("expected %#v, found %#v", tc.expected, out)
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
			t.Errorf("expected err == nil, found: %#v", err)
		}

		var out any
		if err := c.Get(tc.key, &out); err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}

		if out != tc.expected {
			t.Errorf("expected %#v, found %#v", tc.expected, out)
		}
	}
}

type unmarshalerField int

func (f *unmarshalerField) UnmarshalText(data []byte) error {
	str := string(data)
	if str == "valid" {
		*f = 100
		return nil
	}

	return fmt.Errorf("'unmarshalerField' only accepts 'valid' keyword. Got %q instead", str)
}

func TestGet(t *testing.T) {
	type person struct {
		Name          string `json:"name"`
		Age           int    `json:"age"`
		CaseSensitive string `json:"caseSensitive"`
	}

	type unmarshalerPerson struct {
		Name        string           `json:"name"`
		Age         int              `json:"age"`
		Unmarshaler unmarshalerField `json:"unmarshaler"`
	}

	type unmarshalerSubObject struct {
		Person unmarshalerPerson `json:"person"`
	}

	t.Run("decode to no pointer", func(t *testing.T) {
		c, err := setupWithFile([]byte(`{ "foo": "bar" }`))

		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}

		var p person
		err = c.Get("foo", p)

		if err == nil {
			t.Errorf("expected err != nil, found: %#v", err)
		}
	})

	t.Run("decode to nil pointer", func(t *testing.T) {
		c, err := setupWithFile([]byte(`{ "foo": "bar" }`))

		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}

		var p person
		err = c.Get("foo", p)

		if err == nil {
			t.Errorf("expected err != nil, found: %#v", err)
		}
	})

	t.Run("decode partial to non-zero struct", func(t *testing.T) {
		data := `{"person":{"name":"Josh"}}`
		c, err := setupWithFile([]byte(data))
		if err != nil {
			t.Errorf("setting up file expected err == nil, found: %#v", err)
		}

		p := person{Age: 20}
		err = c.Get("person", &p)
		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}

		expected := person{Age: 20, Name: "Josh"}
		if !reflect.DeepEqual(p, expected) {
			t.Errorf("expected %#v when decoding %s, found %#v instead", expected, data, p)
		}
	})

	t.Run("decode from config file to pointer", func(t *testing.T) {
		data := `{
			"foo": {
				"name": "jon",
				"age": 5,
				"caseSensitive": "some"
			}
		}`
		expected := person{Name: "jon", Age: 5, CaseSensitive: "some"}

		c, err := setupWithFile([]byte(data))

		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}

		p := new(person)
		err = c.Get("foo", p)

		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}
		if !reflect.DeepEqual(*p, expected) {
			t.Errorf("expected p == %#v, found: %#v", expected, p)
		}
	})

	t.Run("decode env var to pointer", func(t *testing.T) {
		data := `{
			"name": "jon",
			"age": 5
		}`
		// TODO: The data above should have `"caseSensitive": "some"`, but we currently
		// don't have support for case sensitive env vars...
		var expected person
		_ = json.Unmarshal([]byte(data), &expected)

		t.Setenv("MGC_FOO", data)

		c := setupWithoutFile()

		p := new(person)
		err := c.Get("foo", p)

		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}
		if !reflect.DeepEqual(*p, expected) {
			t.Errorf("expected p == %#v, found: %#v", expected, *p)
		}
	})

	t.Run("decode string in config file to string", func(t *testing.T) {
		data := `{ "foo": "bar" }`
		expected := "bar"

		c, err := setupWithFile([]byte(data))
		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}

		var p string
		err = c.Get("foo", &p)

		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}
		if p != expected {
			t.Errorf("expected p == bar, found: %#v", p)
		}
	})

	t.Run("decode string in env var to string", func(t *testing.T) {
		expected := "bar"
		t.Setenv("MGC_FOO", expected)

		c := setupWithoutFile()

		var p string
		err := c.Get("foo", &p)

		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}
		if p != expected {
			t.Errorf("expected p == %#v, found: %#v", expected, p)
		}
	})

	t.Run("decode object in config file to struct", func(t *testing.T) {
		data := `{
			"foo": {
				"name": "jon",
				"age": 5,
				"caseSensitive": "some"
			}
		}`
		expected := person{Name: "jon", Age: 5, CaseSensitive: "some"}

		c, err := setupWithFile([]byte(data))

		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}

		var p person
		err = c.Get("foo", &p)

		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}
		if !reflect.DeepEqual(p, expected) {
			t.Errorf("expected p == %#v, found: %#v", expected, p)
		}
	})

	t.Run("decode object in env var to struct", func(t *testing.T) {
		data := `{
			"name": "jon",
			"age": 5
		}`
		// TODO: The data above should have `"caseSensitive": "some"`, but we currently
		// don't have support for case sensitive env vars...
		var expected person
		_ = json.Unmarshal([]byte(data), &expected)

		t.Setenv("MGC_FOO", data)

		c := setupWithoutFile()

		var p person
		err := c.Get("foo", &p)

		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}
		if !reflect.DeepEqual(p, expected) {
			t.Errorf("expected p == %#v, found: %#v", expected, p)
		}
	})

	t.Run("decode string in config file to any", func(t *testing.T) {
		data := `{ "foo": "bar" }`
		expected := "bar"

		c, err := setupWithFile([]byte(data))
		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}

		var p any
		err = c.Get("foo", &p)

		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}
		if p != expected {
			t.Errorf("expected p == bar, found: %#v", p)
		}
	})

	t.Run("decode object in config file with unmarshaler types", func(t *testing.T) {
		// We save objects in Config File as strings...
		data := `{
			"foo": "{\"name\":\"jon\",\"age\":5,\"unmarshaler\":\"valid\"}"
		}`

		expected := unmarshalerPerson{
			Name:        "jon",
			Age:         5,
			Unmarshaler: 100,
		}

		c, err := setupWithFile([]byte(data))

		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}

		p := new(unmarshalerPerson)
		err = c.Get("foo", p)

		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}
		if !reflect.DeepEqual(*p, expected) {
			t.Errorf("expected p == %#v, found: %#v", expected, p)
		}
	})

	t.Run("decode object in config file with subfield unmarshaler types", func(t *testing.T) {
		// We save objects in Config File as strings...
		data := `{
			"foo": "{\"person\":{\"name\":\"jon\",\"age\":5,\"unmarshaler\":\"valid\"}}"
		}`

		expected := unmarshalerSubObject{
			Person: unmarshalerPerson{
				Name:        "jon",
				Age:         5,
				Unmarshaler: 100,
			},
		}

		c, err := setupWithFile([]byte(data))

		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}

		p := new(unmarshalerSubObject)
		err = c.Get("foo", p)

		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}
		if !reflect.DeepEqual(*p, expected) {
			t.Errorf("expected p == %#v, found: %#v", expected, p)
		}
	})
	t.Run("decode object in config file to any", func(t *testing.T) {
		// We save objects in Config File as strings...
		data := `{
			"foo": "{\"name\":\"jon\",\"age\":5,\"caseSensitive\":\"some\"}"
		}`

		expected := map[string]any{
			"name":          "jon",
			"age":           float64(5),
			"caseSensitive": "some",
		}

		c, err := setupWithFile([]byte(data))

		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}

		var p any
		err = c.Get("foo", &p)

		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}
		if !reflect.DeepEqual(p, expected) {
			t.Errorf("expected p == %#v, found: %#v", expected, p)
		}
	})

	t.Run("decode string in env var to any", func(t *testing.T) {
		expected := "bar"

		t.Setenv("MGC_FOO", expected)

		c := setupWithoutFile()

		var p any
		err := c.Get("foo", &p)

		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}
		if p != expected {
			t.Errorf("expected p == %#v, found: %#v", expected, p)
		}
	})

	t.Run("decode object in env var to any", func(t *testing.T) {
		data := `{
			"name": "jon",
			"age": 5,
			"caseSensitive": "some"
		}`

		t.Setenv("MGC_FOO", data)

		expected := map[string]any{
			"name":          "jon",
			"age":           float64(5),
			"caseSensitive": "some",
		}

		c := setupWithoutFile()

		var p any
		err := c.Get("foo", &p)

		if err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}
		if !reflect.DeepEqual(p, expected) {
			t.Errorf("expected p == %#v, found: %#v", expected, p)
		}
	})
}

func TestSetWithoutFile(t *testing.T) {
	tests := []test{
		{key: "foo", fileData: []byte{}, expected: "woo"},
	}

	for _, tc := range tests {
		c := setupWithoutFile()

		if err := c.Set(tc.key, tc.expected); err != nil {
			t.Errorf("expected err == nil , found %#v", err)
		}

		var v any
		if err := c.Get(tc.key, &v); err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}
		if v != tc.expected {
			t.Errorf("expected %#v, found %#v", tc.expected, v)
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
			t.Errorf("expected err == nil, found: %#v", err)
		}

		if err := c.Set(tc.key, tc.expected); err != nil {
			t.Errorf("expected err == nil , found %#v", err)
		}

		var v any
		if err := c.Get(tc.key, &v); err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}
		if v != tc.expected {
			t.Errorf("expected %#v, found %#v", tc.expected, v)
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
			t.Errorf("expected err == nil, found %#v", err)
		}

		var v any
		if err := c.Get(tc.key, &v); err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}
		if v != tc.expected {
			t.Errorf("expected %#v, found %#v", tc.expected, v)
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
			t.Errorf("expected err == nil, found: %#v", err)
		}

		if err := c.Delete(tc.key); err != nil {
			t.Errorf("expected err == nil, found %#v", err)
		}

		var v any
		if err := c.Get(tc.key, &v); err != nil {
			t.Errorf("expected err == nil, found: %#v", err)
		}
		if v != tc.expected {
			t.Errorf("expected %#v, found %#v", tc.expected, v)
		}
	}
}
