package cmd

import (
	"strings"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/profusion/magalu/libs/parser"
)

func SanitizeFlagName(name string) string {
	sanitizedName := strings.ToLower(name)
	sanitizedName = strings.ReplaceAll(sanitizedName, " ", "-")

	return sanitizedName
}

func isPathParam(param *openapi3.Parameter) bool {
	return param.In == "path"
}

func isHeatherParam(param *openapi3.Parameter) bool {
	return param.In == "header"
}

func isRequiredProperty(requriedSet []string, property string) bool {
	for _, req := range requriedSet {
		if req == SanitizeFlagName(property) {
			return true
		}
	}

	return false
}

func GetParams(params openapi3.Parameters) ([]*parser.Param, []*parser.Param) {
	pathParams := []*parser.Param{}
	headerParams := []*parser.Param{}

	for _, paramRef := range params {
		param := parser.Param{
			Type:        paramRef.Value.Schema.Value.Type,
			Name:        paramRef.Value.Name,
			Required:    paramRef.Value.Required,
			Description: paramRef.Value.Description,
		}

		if isPathParam(paramRef.Value) {
			pathParams = append(pathParams, &param)
		}

		if isHeatherParam(paramRef.Value) {
			headerParams = append(headerParams, &param)
		}
	}

	return pathParams, headerParams
}

func GetRequestBodyParams(action *parser.OpenAPIAction) []*parser.Param {

	reqbody := []*parser.Param{}

	request := action.Request
	if request == nil {
		return reqbody
	}

	content := request.Value.Content.Get("application/json").Schema.Value
	requiredProperties := content.Required

	for _, propertyRef := range content.Properties {
		property := propertyRef.Value

		sanitizedName := SanitizeFlagName(property.Title)
		param := parser.Param{
			Type:        property.Type,
			Name:        sanitizedName,
			Required:    isRequiredProperty(requiredProperties, sanitizedName),
			Description: property.Description,
		}

		reqbody = append(reqbody, &param)
	}

	return reqbody
}
