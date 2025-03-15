package main

import (
	"fmt"
	"reflect"
	"strings"
	"text/template"

	"ella.to/ella/internal/strcase"
)

func Generate(pkg, output string, docs []*Document) error {
	mainDoc := &Document{
		Consts:   make([]*Const, 0),
		Enums:    make([]*Enum, 0),
		Models:   make([]*Model, 0),
		Services: make([]*Service, 0),
		Errors:   make([]*CustomError, 0),
	}

	for _, doc := range docs {
		for _, c := range doc.Consts {
			mainDoc.Consts = append(mainDoc.Consts, c)
		}

		for _, e := range doc.Enums {
			mainDoc.Enums = append(mainDoc.Enums, e)
		}

		for _, m := range doc.Models {
			mainDoc.Models = append(mainDoc.Models, m)
		}

		for _, s := range doc.Services {
			mainDoc.Services = append(mainDoc.Services, s)
		}

		for _, e := range doc.Errors {
			mainDoc.Errors = append(mainDoc.Errors, e)
		}
	}

	if strings.HasSuffix(output, ".go") {
		return generateGo(pkg, output, mainDoc)
	} else if strings.HasSuffix(output, ".ts") {
		return generateTypescript(pkg, output, mainDoc)
	}

	return fmt.Errorf("unknown output file type: %s", output)
}

var defaultFuncsMap = template.FuncMap{
	"ToLower":      strings.ToLower,
	"ToUpper":      strings.ToUpper,
	"ToPascalCase": strcase.ToPascal,
	"ToCamelCase":  strcase.ToCamel,
	"ToSnakeCase":  strcase.ToSnake,
	"Length": func(v any) int {
		switch val := v.(type) {
		case string:
			return len(val)
		case []interface{}:
			return len(val)
		case map[string]interface{}:
			return len(val)
		default:
			// Handle other types that implement len
			// For example, slices, arrays, maps, etc.
			if rv := reflect.ValueOf(v); rv.Kind() == reflect.Slice ||
				rv.Kind() == reflect.Array ||
				rv.Kind() == reflect.Map ||
				rv.Kind() == reflect.Chan ||
				rv.Kind() == reflect.String {
				return rv.Len()
			}
			return 0
		}
	},
}

func mapperFunc[I, O any](list []I, f func(I) O) []O {
	var results []O

	for _, item := range list {
		results = append(results, f(item))
	}

	return results
}

func filterFunc[I any](list []I, f func(I) bool) []I {
	var results []I

	for _, item := range list {
		if f(item) {
			results = append(results, item)
		}
	}

	return results
}

func createIsModelTypeFunc(models []*Model) func(value string) bool {
	set := make(map[string]struct{})
	for _, model := range models {
		set[model.Name.Token.Value] = struct{}{}
	}

	return func(value string) bool {
		_, ok := set[value]
		return ok
	}
}
