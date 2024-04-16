package code

import (
	"strings"
	"text/template"

	"ella.to/ella/pkg/strcase"
)

var DefaultFuncsMap = template.FuncMap{
	"ToLower":      strings.ToLower,
	"ToUpper":      strings.ToUpper,
	"ToPascalCase": strcase.ToPascal,
	"ToCamelCase":  strcase.ToCamel,
	"ToSnakeCase":  strcase.ToSnake,
}
