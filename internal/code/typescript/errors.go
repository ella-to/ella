package typescript

import (
	"sort"

	"ella.to/ella/internal/ast"
	"ella.to/ella/internal/ast/astutil"
	"ella.to/ella/pkg/sliceutil"
)

type CustomError struct {
	Name string
	Code int64
}

type CustomErrors []CustomError

func (c *CustomErrors) Parse(prog *ast.Program) error {
	*c = sliceutil.Mapper(astutil.GetCustomErrors(prog), func(customError *ast.CustomError) CustomError {
		return CustomError{
			Name: customError.Name.String(),
			Code: customError.Code,
		}
	})

	sort.Slice(*c, func(i, j int) bool {
		return (*c)[i].Name < (*c)[j].Name
	})

	return nil
}
