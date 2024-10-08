package golang

import (
	"strings"

	"ella.to/ella/internal/ast"
	"ella.to/ella/internal/ast/astutil"
	"ella.to/ella/pkg/sliceutil"
	"ella.to/ella/pkg/strcase"
)

type ModelField struct {
	Name string
	Type string
	Tags string
}

type ModelFields []ModelField

func (m *ModelFields) Parse(message *ast.Model, isModelType func(value string) bool) error {
	*m = sliceutil.Mapper(message.Fields, func(field *ast.Field) ModelField {
		typ := parseType(field.Type, isModelType)
		return ModelField{
			Name: field.Name.String(),
			Type: typ,
			Tags: parseModelFieldOptions(field),
		}
	})
	return nil
}

type Model struct {
	Name   string
	Fields ModelFields
}

type Models []Model

func (m *Models) Parse(prog *ast.Program) error {
	isModelType := astutil.CreateIsModelTypeFunc(astutil.GetModels(prog))

	*m = sliceutil.Mapper(astutil.GetModels(prog), func(message *ast.Model) Model {
		msg := Model{
			Name: message.Name.String(),
		}

		msg.Fields.Parse(message, isModelType)

		return msg
	})

	return nil
}

func parseModelFieldOptions(field *ast.Field) string {
	var sb strings.Builder

	mapper := make(map[string]ast.Value)
	for _, opt := range field.Options {
		mapper[strings.ToLower(opt.Name.Token.Literal)] = opt.Value
	}

	jsonTagValue := strings.ToLower(strcase.ToSnake(field.Name.String()))

	jsonValue, ok := mapper["json"]
	if ok {
		switch jsonValue := jsonValue.(type) {
		case *ast.ValueString:
			jsonTagValue = jsonValue.Token.Literal
		case *ast.ValueBool:
			if !jsonValue.Value {
				jsonTagValue = "-"
			}
		}
	}

	jsonOmitEmptyValue, ok := mapper["jsonomitempty"]
	if ok && jsonTagValue != "-" {
		switch value := jsonOmitEmptyValue.(type) {
		case *ast.ValueBool:
			if value.Value {
				jsonTagValue += ",omitempty"
			}
		}
	}

	sb.WriteString(`json:"`)
	sb.WriteString(jsonTagValue)
	sb.WriteString(`"`)

	yamlTagValue := strings.ToLower(strcase.ToSnake(field.Name.String()))

	yamlValue, ok := mapper["yaml"]
	if ok {
		switch yamlValue := yamlValue.(type) {
		case *ast.ValueString:
			yamlTagValue = yamlValue.Token.Literal
		case *ast.ValueBool:
			if !yamlValue.Value {
				yamlTagValue = "-"
			}
		}
	}

	sb.WriteString(` yaml:"`)
	sb.WriteString(yamlTagValue)
	sb.WriteString(`"`)

	return sb.String()
}
