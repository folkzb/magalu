package utils

import (
	"bytes"
	"strings"
	"text/template"

	"golang.org/x/exp/slices"
)

var finishedTemplateStrings = []string{
	"finished",
	"terminated",
	"true",
}

func NewTemplate(expression string) (tmpl *template.Template, err error) {
	return NewTemplateFilename(expression, "<expression>")
}

func NewTemplateFilename(expression string, fileName string) (tmpl *template.Template, err error) {
	return template.New(fileName).Parse(expression)
}

func CreateTemplateChecker(expression string) (checker func(document any) (bool, error), err error) {
	jp, err := NewTemplate(expression)
	if err != nil {
		return nil, err
	}
	return CreateTemplateCheckerFromTemplate(jp), nil
}

func CreateTemplateCheckerFromTemplate(tmpl *template.Template) (checker func(document any) (bool, error)) {
	return func(value any) (ok bool, err error) {
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, value)
		if err != nil {
			return false, err
		}
		s := buf.String()
		s = strings.Trim(s, " \t\n\r")
		return slices.Contains(finishedTemplateStrings, s), nil
	}
}
