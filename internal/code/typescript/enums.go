package typescript

import (
	"ella.to/ella/internal/ast"
	"ella.to/ella/internal/ast/astutil"
	"ella.to/ella/pkg/sliceutil"
	"ella.to/ella/pkg/strcase"
)

type EnumKeyValue struct {
	Name  string
	Value string
}

type Enum struct {
	Name string
	Keys []EnumKeyValue
}

type Enums []Enum

func (e *Enums) Parse(prog *ast.Program) error {
	*e = sliceutil.Mapper(astutil.GetEnums(prog), func(enum *ast.Enum) Enum {
		return Enum{
			Name: enum.Name.String(),
			Keys: sliceutil.Mapper(sliceutil.Filter(enum.Sets, func(set *ast.EnumSet) bool {
				return set.Name.String() != "_"
			}), func(set *ast.EnumSet) EnumKeyValue {
				return EnumKeyValue{
					Name:  set.Name.String(),
					Value: strcase.ToSnake(set.Name.String()),
				}
			}),
		}
	})

	return nil
}
