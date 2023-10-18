package http

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Can be safely Marshaled/Unmarshaled with JSON, unlike regular impl
type MarshalableRequest http.Request

func (r *MarshalableRequest) MarshalJSON() ([]byte, error) {
	b := &bytes.Buffer{} // holds serialized representation
	err := (*http.Request)(r).Write(b)
	if err != nil {
		return nil, err
	}
	return json.Marshal(b.String())
}

func (r *MarshalableRequest) UnmarshalJSON(data []byte) error {
	var str string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	unmarshaled, err := http.ReadRequest(bufio.NewReader(strings.NewReader(str)))
	if err != nil {
		return fmt.Errorf("error unmarshaling http request object: %w", err)
	}
	*r = *(*MarshalableRequest)(unmarshaled)
	return nil
}

// Can be safely Marshaled/Unmarshaled with JSON, unlike regular impl
type MarshalableResponse http.Response

func (r *MarshalableResponse) MarshalJSON() ([]byte, error) {
	b := &bytes.Buffer{} // holds serialized representation
	err := (*http.Response)(r).Write(b)
	if err != nil {
		return nil, err
	}
	return json.Marshal(b.String())
}

func (r *MarshalableResponse) UnmarshalJSON(data []byte) error {
	var str string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	unmarshaled, err := http.ReadResponse(bufio.NewReader(strings.NewReader(str)), nil)
	if err != nil {
		return fmt.Errorf("error unmarshaling http response object: %w", err)
	}
	*r = *(*MarshalableResponse)(unmarshaled)
	return nil
}
