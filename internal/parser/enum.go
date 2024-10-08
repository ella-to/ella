package parser

import (
	"strconv"
	"strings"

	"ella.to/ella/internal/ast"
	"ella.to/ella/internal/token"
	"ella.to/ella/pkg/strcase"
)

func ParseEnum(p *Parser) (enum *ast.Enum, err error) {
	if p.Peek().Type != token.Enum {
		return nil, p.WithError(p.Peek(), "expected 'enum' keyword")
	}

	enum = &ast.Enum{Token: p.Next()}

	if p.Peek().Type != token.Identifier {
		return nil, p.WithError(p.Peek(), "expected identifier for defining an enum")
	}

	nameTok := p.Next()

	if !strcase.IsPascal(nameTok.Literal) {
		return nil, p.WithError(nameTok, "enum name must be in Pascal Case format")
	}

	enum.Name = &ast.Identifier{Token: nameTok}

	if p.Peek().Type != token.OpenCurly {
		return nil, p.WithError(p.Peek(), "expected '{' after enum declaration")
	}

	p.Next() // skip '{'

	for p.Peek().Type != token.CloseCurly {
		set, err := parseEnumSet(p)
		if err != nil {
			return nil, err
		}

		enum.Sets = append(enum.Sets, set)
	}

	p.Next() // skip '}'

	// we corrected the values

	var next int64
	var minV int64
	var maxV int64

	for _, set := range enum.Sets {
		if set.Defined {
			next = set.Value.Value + 1
			continue
		}

		set.Value = &ast.ValueInt{
			Token:   nil,
			Value:   next,
			Defined: false,
		}

		minV = min(minV, next)
		maxV = max(maxV, next)

		next++
	}

	enum.Size = getIntSize(minV, maxV)

	for _, set := range enum.Sets {
		set.Value.Size = enum.Size
	}

	return enum, nil
}

func parseEnumSet(p *Parser) (*ast.EnumSet, error) {
	if p.Peek().Type != token.Identifier {
		return nil, p.WithError(p.Peek(), "expected identifier for defining an enum constant")
	}

	nameTok := p.Next()

	if nameTok.Literal != "_" && !strcase.IsPascal(nameTok.Literal) {
		return nil, p.WithError(nameTok, "enum's set name must be in Pascal Case format")
	}

	if p.Peek().Type != token.Assign {
		return &ast.EnumSet{
			Name: &ast.Identifier{Token: nameTok},
			Value: &ast.ValueInt{
				Value: 0,
			},
		}, nil
	}

	p.Next() // skip '='

	if p.Peek().Type != token.ConstInt {
		return nil, p.WithError(p.Peek(), "expected constant integer value for defining an enum set value")
	}

	valueTok := p.Next()
	value, err := strconv.ParseInt(strings.ReplaceAll(valueTok.Literal, "_", ""), 10, 64)
	if err != nil {
		return nil, p.WithError(valueTok, "invalid integer value for defining an enum constant value: ", err)
	}

	return &ast.EnumSet{
		Name: &ast.Identifier{Token: nameTok},
		Value: &ast.ValueInt{
			Token:   valueTok,
			Value:   value,
			Defined: true,
		},
		Defined: true,
	}, nil
}
