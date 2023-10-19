package blueprint

import (
	"fmt"
	"text/template"

	"golang.org/x/exp/maps"
	"magalu.cloud/core/utils"
)

type checkError struct {
	OriginalErr error
	Document    map[string]any
	Message     string
	Check       *checkSpec
}

func (c checkError) Error() string {
	return c.Message
}

func (c checkError) Unwrap() error {
	return c.OriginalErr
}

var _ error = (*checkError)(nil)

type checkSpec struct {
	conditionSpec

	ErrorMessageTemplate string `json:"errorMessageTemplate,omitempty"`
	tmpl                 *template.Template
}

func (c *checkSpec) wrapError(origError error, jsonPathDocument map[string]any) *checkError {
	msg := ""

	if c.tmpl != nil {
		doc := maps.Clone(jsonPathDocument)
		doc["error"] = origError
		doc["error_message"] = origError.Error()
		msg, _ = utils.ExecuteTemplateTrimmed(c.tmpl, doc)
	}

	if msg == "" {
		msg = origError.Error()
	}

	return &checkError{origError, jsonPathDocument, msg, c}
}

func (c *checkSpec) check(jsonPathDocument map[string]any) (err error) {
	if c == nil {
		return nil
	}

	var ok bool
	if ok, err = c.conditionSpec.check(jsonPathDocument); ok {
		return nil
	}

	if err == nil {
		err = fmt.Errorf("failed condition: %s", c.conditionSpec.String())
	}

	return c.wrapError(err, jsonPathDocument)
}

func (c *checkSpec) validate() (err error) {
	if c == nil {
		return nil
	}

	err = c.conditionSpec.validate()
	if err != nil {
		return err
	}

	if c.ErrorMessageTemplate != "" {
		c.tmpl, err = utils.NewTemplate(c.ErrorMessageTemplate)
		if err != nil {
			return err
		}
	}

	return nil
}
