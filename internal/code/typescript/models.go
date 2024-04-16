package typescript

import (
	"ella.to/ella/internal/ast"
	"ella.to/ella/internal/ast/astutil"
	"ella.to/ella/pkg/sliceutil"
	"ella.to/ella/pkg/strcase"
)

type ModelField struct {
	Name string
	Type string
}

type ModelFields []ModelField

func (m *ModelFields) Parse(message *ast.Model) error {
	*m = sliceutil.Filter(sliceutil.Mapper(message.Fields, func(field *ast.Field) ModelField {
		name := strcase.ToSnake(field.Name.String())
		for _, opt := range field.Options {
			if opt.Name.String() == "Json" {
				switch v := opt.Value.(type) {
				case *ast.ValueString:
					name = v.TokenLiteral()
				case *ast.ValueBool:
					if !v.Value {
						name = ""
					}
				}
				break
			}
		}

		return ModelField{
			Name: name,
			Type: parseType(field.Type),
		}
	}), func(field ModelField) bool {
		return field.Name != ""
	})
	return nil
}

type Model struct {
	Name   string
	Fields ModelFields
}

type Models []Model

func (m *Models) Parse(prog *ast.Program) error {
	*m = sliceutil.Mapper(astutil.GetModels(prog), func(message *ast.Model) Model {
		msg := Model{
			Name: message.Name.String(),
		}

		msg.Fields.Parse(message)

		return msg
	})

	return nil
}
