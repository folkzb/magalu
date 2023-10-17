package utils

import "text/template"

func NewTemplate(expression string) (tmpl *template.Template, err error) {
	return NewTemplateFilename(expression, "<expression>")
}

func NewTemplateFilename(expression string, fileName string) (tmpl *template.Template, err error) {
	return template.New(fileName).Parse(expression)
}
