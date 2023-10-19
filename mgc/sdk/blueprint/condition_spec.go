package blueprint

import (
	"errors"
	"fmt"
	"text/template"

	"github.com/PaesslerAG/gval"
	"magalu.cloud/core/utils"
)

type conditionSpec struct {
	JSONPathQuery string `json:"jsonPathQuery,omitempty"`
	TemplateQuery string `json:"templateQuery,omitempty"`

	jsonPathQuery gval.Evaluable
	templateQuery *template.Template
	checker       func(doc any) (bool, error)
}

func (c *conditionSpec) String() string {
	if c.JSONPathQuery != "" {
		return fmt.Sprintf("jsonPathQuery: %q", c.JSONPathQuery)
	}
	if c.TemplateQuery != "" {
		return fmt.Sprintf("templateQuery: %q", c.TemplateQuery)
	}
	return "no conditions specified"
}

func (c *conditionSpec) check(doc any) (bool, error) {
	return c.checker(doc)
}

func (c *conditionSpec) validate() (err error) {
	if c.JSONPathQuery == "" && c.TemplateQuery == "" {
		return errors.New("expected one of jsonPathQuery or templateQuery")
	} else if c.JSONPathQuery != "" && c.TemplateQuery != "" {
		return errors.New("cannot specify both jsonPathQuery and templateQuery")
	}

	if c.JSONPathQuery != "" {
		c.jsonPathQuery, err = utils.NewJsonPath(c.JSONPathQuery)
		if err != nil {
			return fmt.Errorf("jsonPathQuery: %w", err)
		}
	}

	if c.TemplateQuery != "" {
		c.templateQuery, err = utils.NewTemplate(c.TemplateQuery)
		if err != nil {
			return fmt.Errorf("templateQuery: %w", err)
		}
	}

	if c.checker == nil {
		if c.jsonPathQuery != nil {
			c.checker = utils.CreateJsonPathCheckerFromEvaluable(c.jsonPathQuery)
		} else if c.templateQuery != nil {
			c.checker = utils.CreateTemplateCheckerFromTemplate(c.templateQuery)
		}
	}
	return nil
}
