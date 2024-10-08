package parser

import (
	"ella.to/ella/internal/ast"
	"ella.to/ella/internal/token"
	"ella.to/ella/pkg/strcase"
)

func ParseConst(p *Parser) (*ast.Const, error) {
	if p.Peek().Type != token.Const {
		return nil, p.WithError(p.Peek(), "expected 'const' keyword")
	}

	tok := p.Next()

	if p.Peek().Type != token.Identifier {
		return nil, p.WithError(p.Peek(), "expected identifier for defining a constant")
	}

	nameTok := p.Next()

	if !strcase.IsPascal(nameTok.Literal) {
		return nil, p.WithError(nameTok, "constant name must be in PascalCase format")
	}

	if p.Peek().Type != token.Assign {
		return nil, p.WithError(p.Peek(), "expected '=' after an identifier for defining a constant")
	}

	p.Next() // skip '='

	value, err := ParseValue(p)
	if err != nil {
		return nil, err
	}

	return &ast.Const{
		Token: tok,
		Name:  &ast.Identifier{Token: nameTok},
		Value: value,
	}, nil
}
