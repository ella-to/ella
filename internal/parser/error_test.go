package parser_test

import (
	"testing"

	"ella.to/ella/internal/ast"
	"ella.to/ella/internal/parser"
)

func TestParseCustomError(t *testing.T) {
	testCases := TestCases{
		{
			Input:  `error ErrUserNotFound { Code = 1000 HttpStatus = NotFound Msg = "user not found" }`,
			Output: `error ErrUserNotFound { Code = 1000 HttpStatus = NotFound Msg = "user not found" }`,
		},
		{
			Input:  "error ErrUserNotFound { Code = 1000 HttpStatus = NotFound Msg = `user not found` }",
			Output: "error ErrUserNotFound { Code = 1000 HttpStatus = NotFound Msg = `user not found` }",
		},
		{
			Input:  "error ErrUserNotFound { HttpStatus = NotFound Msg = `user not found` }",
			Output: "error ErrUserNotFound { Code = 0 HttpStatus = NotFound Msg = `user not found` }",
		},
	}

	runTests(t, func(p *parser.Parser) (ast.Node, error) {
		return parser.ParseCustomError(p)
	}, testCases)
}
