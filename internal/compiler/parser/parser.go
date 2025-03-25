package parser

import (
	"math"
	"strconv"
	"strings"

	"ella.to/ella/internal/compiler/ast"
	"ella.to/ella/internal/compiler/scanner"
	"ella.to/ella/internal/compiler/token"
	"ella.to/ella/internal/strcase"
)

type Parser struct {
	tokens   token.Iterator
	nextTok  *token.Token
	currTok  *token.Token
	comments []*ast.Comment
}

func (p *Parser) Current() *token.Token {
	return p.currTok
}

func (p *Parser) Next() *token.Token {
	if p.nextTok != nil {
		p.currTok = p.nextTok
		p.nextTok = nil
	} else {
		p.currTok = p.tokens.NextToken()
	}

	return p.currTok
}

func (p *Parser) Peek() *token.Token {
	if p.nextTok == nil {
		p.nextTok = p.tokens.NextToken()
	}

	return p.nextTok
}

func NewParser(input string) *Parser {
	tokenEmitter := token.NewEmitterIterator()
	go scanner.Start(tokenEmitter, scanner.Lex, input)
	return &Parser{
		tokens: tokenEmitter,
	}
}

func NewWithFilenames(filenames ...string) *Parser {
	tokenEmitter := token.NewEmitterIterator()
	go scanner.StartWithFilenames(tokenEmitter, scanner.Lex, filenames...)
	return &Parser{
		tokens: tokenEmitter,
	}
}

// Parse Comment

func ParseComment(p *Parser) (*ast.Comment, error) {
	if p.Peek().Type != token.Comment {
		return nil, NewError(p.Peek(), "expected comment but got %s", p.Peek().Type)
	}

	return &ast.Comment{Token: p.Next()}, nil
}

// Parse Contsnant

func ParseConst(p *Parser) (*ast.Const, error) {
	if p.Peek().Type != token.Const {
		return nil, NewError(p.Peek(), "expected const, got %s", p.Peek().Type)
	}

	constant := &ast.Const{Token: p.Next()}

	if p.Peek().Type != token.Identifier {
		return nil, NewError(p.Peek(), "expected identifier after const keyword, got %s", p.Peek().Type)
	}

	constant.Identifier = &ast.Identifier{Token: p.Next()}

	if p.Peek().Type != token.Assign {
		return nil, NewError(p.Peek(), "expected = after identifier, got %s", p.Peek().Type)
	}

	p.Next()

	value, err := ParseValue(p)
	if err != nil {
		return nil, err
	}

	constant.Value = value

	return constant, nil
}

// Parse Enum

func ParseEnum(p *Parser) (enum *ast.Enum, err error) {
	if p.Peek().Type != token.Enum {
		return nil, NewError(p.Peek(), "expected 'enum' keyword")
	}

	enum = &ast.Enum{Token: p.Next()}

	if p.Peek().Type != token.Identifier {
		return nil, NewError(p.Peek(), "expected identifier for defining an enum")
	}

	nameTok := p.Next()

	if !strcase.IsPascal(nameTok.Value) {
		return nil, NewError(nameTok, "enum name must be in Pascal Case format")
	}

	enum.Name = &ast.Identifier{Token: nameTok}

	if p.Peek().Type != token.OpenCurly {
		return nil, NewError(p.Peek(), "expected '{' after enum declaration")
	}

	p.Next() // skip '{'

	for {
		peek := p.Peek()

		if peek.Type == token.CloseCurly {
			break
		}

		if peek.Type == token.Identifier {
			set, err := EnumSet(p)
			if err != nil {
				return nil, err
			}

			set.AddComments(p.comments...)
			p.comments = p.comments[:0]

			enum.Sets = append(enum.Sets, set)
			continue
		}

		if peek.Type == token.Comment {
			comment, err := ParseComment(p)
			if err != nil {
				return nil, err
			}

			comment.Position = ast.CommentBottom
			p.comments = append(p.comments, comment)
			continue
		}
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

	for _, comment := range p.comments {
		enum.AddComments(comment)
	}

	p.comments = p.comments[:0]

	return enum, nil
}

func EnumSet(p *Parser) (*ast.EnumSet, error) {
	if p.Peek().Type != token.Identifier {
		return nil, NewError(p.Peek(), "expected identifier for defining an enum constant")
	}

	nameTok := p.Next()

	if nameTok.Value != "_" && !strcase.IsPascal(nameTok.Value) {
		return nil, NewError(nameTok, "enum's set name must be in Pascal Case format")
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
		return nil, NewError(p.Peek(), "expected constant integer value for defining an enum set value")
	}

	valueTok := p.Next()
	value, err := strconv.ParseInt(strings.ReplaceAll(valueTok.Value, "_", ""), 10, 64)
	if err != nil {
		return nil, NewError(valueTok, "invalid integer value for defining an enum constant value: %s", err)
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

// Parse Option

func ParseOption(p *Parser) (option *ast.Option, err error) {
	if p.Peek().Type != token.Identifier {
		return nil, NewError(p.Peek(), "expected identifier for defining a message field option")
	}

	nameTok := p.Next()

	option = &ast.Option{
		Name: &ast.Identifier{Token: nameTok},
	}

	if p.Peek().Type != token.Assign {
		option.Value = &ast.ValueBool{
			Token:       nil,
			Value:       true,
			UserDefined: false,
		}

		return option, nil
	}

	p.Next() // skip '='

	option.Value, err = ParseValue(p)
	if err != nil {
		return nil, err
	}

	return option, nil
}

func ParseOptions(p *Parser) (*ast.Options, error) {
	options := &ast.Options{
		List:     make([]*ast.Option, 0),
		Comments: make([]*ast.Comment, 0),
	}

	p.Next() // skip '{'

	for {
		peek := p.Peek()

		if peek.Type == token.CloseCurly {
			break
		}

		if peek.Type == token.Comment {
			comment, err := ParseComment(p)
			if err != nil {
				return nil, err
			}

			p.comments = append(p.comments, comment)
			continue
		}

		option, err := ParseOption(p)
		if err != nil {
			return nil, err
		}

		if len(p.comments) > 0 {
			option.AddComments(p.comments...)
			p.comments = p.comments[:0]
		}

		options.List = append(options.List, option)
	}

	p.Next() // skip '}'

	if len(p.comments) > 0 {
		for _, comment := range p.comments {
			comment.Position = ast.CommentBottom
		}

		options.AddComments(p.comments...)
		p.comments = p.comments[:0]
	}

	return options, nil
}

// Parse Model

func ParseModel(p *Parser) (*ast.Model, error) {
	if p.Peek().Type != token.Model {
		return nil, NewError(p.Peek(), "expected 'model' keyword")
	}

	model := &ast.Model{Token: p.Next()}

	if p.Peek().Type != token.Identifier {
		return nil, NewError(p.Peek(), "expected identifier for defining a model")
	}

	nameTok := p.Next()

	if !strcase.IsPascal(nameTok.Value) {
		return nil, NewError(nameTok, "model name must be in PascalCase format")
	}

	model.Name = &ast.Identifier{Token: nameTok}

	if p.Peek().Type != token.OpenCurly {
		return nil, NewError(p.Peek(), "expected '{' after model declaration")
	}

	p.Next() // skip '{'

	if len(p.comments) > 0 {
		model.AddComments(p.comments...)
		p.comments = p.comments[:0]
	}

	for {
		peek := p.Peek()

		if peek.Type == token.CloseCurly {
			break
		}

		if peek.Type == token.Comment {
			comment, err := ParseComment(p)
			if err != nil {
				return nil, err
			}

			p.comments = append(p.comments, comment)
			continue
		}

		if peek.Type == token.Extend {
			extend, err := ParseExtend(p)
			if err != nil {
				return nil, err
			}

			if len(p.comments) > 0 {
				extend.AddComments(p.comments...)
				p.comments = p.comments[:0]
			}

			model.Extends = append(model.Extends, extend)
			continue
		}

		field, err := ParseModelField(p)
		if err != nil {
			return nil, err
		}

		model.Fields = append(model.Fields, field)
	}

	p.Next() // skip '}'

	if len(p.comments) > 0 {
		for _, comment := range p.comments {
			comment.Position = ast.CommentBottom
		}

		model.AddComments(p.comments...)
		p.comments = p.comments[:0]
	}

	return model, nil
}

func ParseExtend(p *Parser) (*ast.Extend, error) {
	if p.Peek().Type != token.Extend {
		return nil, NewError(p.Peek(), "expected '...' keyword")
	}

	p.Next() // skip '...'

	if p.Peek().Type != token.Identifier {
		return nil, NewError(p.Peek(), "expected identifier for extending a message")
	}

	nameTok := p.Next()

	if !strcase.IsPascal(nameTok.Value) {
		return nil, NewError(nameTok, "extend message name must be in PascalCase format")
	}

	return &ast.Extend{
		Name:     &ast.Identifier{Token: nameTok},
		Comments: make([]*ast.Comment, 0),
	}, nil
}

func ParseModelField(p *Parser) (field *ast.Field, err error) {
	if p.Peek().Type != token.Identifier {
		return nil, NewError(p.Peek(), "expected identifier for defining a message field")
	}

	nameTok := p.Next()

	if !strcase.IsPascal(nameTok.Value) {
		return nil, NewError(nameTok, "message field name must be in PascalCase format")
	}

	field = &ast.Field{
		Name:     &ast.Identifier{Token: nameTok},
		Options:  &ast.Options{List: make([]*ast.Option, 0)},
		Comments: make([]*ast.Comment, 0),
	}

	peek := p.Peek()

	switch peek.Type {
	case token.Optional:
		field.IsOptional = true
		p.Next() // skip '?'

		if p.Peek().Type != token.Colon {
			return nil, NewError(p.Peek(), "expected ':' after '?'")
		}
		p.Next() // skip ':'

	case token.Colon:
		field.IsOptional = false
		p.Next() // skip ':'
	default:
		return nil, NewError(peek, "expected ':' or '?' after message field name")
	}

	field.Type, err = ParseType(p)
	if err != nil {
		return nil, err
	}

	if len(p.comments) > 0 {
		field.AddComments(p.comments...)
		p.comments = p.comments[:0]
	}

	if p.Peek().Type != token.OpenCurly {
		return field, nil
	}

	field.Options, err = ParseOptions(p)
	if err != nil {
		return nil, err
	}

	return field, nil
}

// Parse Type

func ParseType(p *Parser) (ast.Type, error) {
	peek := p.Peek()

	switch peek.Type {
	case token.Map:
		return ParseMapType(p)
	case token.Array:
		return ParseArrayType(p)
	case token.Bool:
		return &ast.Bool{Token: p.Next()}, nil
	case token.Byte:
		return &ast.Byte{Token: p.Next()}, nil
	case token.Int8, token.Int16, token.Int32, token.Int64:
		tok := p.Next()
		return &ast.Int{
			Token: tok,
			Size:  extractTypeBits("int", tok.Value),
		}, nil
	case token.Uint8, token.Uint16, token.Uint32, token.Uint64:
		tok := p.Next()
		return &ast.Uint{
			Token: tok,
			Size:  extractTypeBits("uint", tok.Value),
		}, nil
	case token.Float32, token.Float64:
		tok := p.Next()
		return &ast.Float{
			Token: tok,
			Size:  extractTypeBits("float", tok.Value),
		}, nil
	case token.Timestamp:
		return &ast.Timestamp{Token: p.Next()}, nil
	case token.String:
		return &ast.String{Token: p.Next()}, nil
	case token.Any:
		return &ast.Any{Token: p.Next()}, nil
	case token.Identifier:
		nameTok := p.Next()

		if !strcase.IsPascal(nameTok.Value) {
			return nil, NewError(nameTok, "custom type name must be in PascalCase format")
		}

		return &ast.CustomType{Token: nameTok}, nil
	default:
		return nil, NewError(peek, "expected type")
	}
}

func ParseMapType(p *Parser) (*ast.Map, error) {
	if p.Peek().Type != token.Map {
		return nil, NewError(p.Peek(), "expected 'map' keyword")
	}

	mapTok := p.Next()

	if p.Peek().Type != token.OpenAngle {
		return nil, NewError(p.Peek(), "expected '<' after 'map' keyword")
	}

	p.Next() // skip '<'

	keyType, err := ParseMapKeyType(p)
	if err != nil {
		return nil, err
	}

	if p.Peek().Type != token.Comma {
		return nil, NewError(p.Peek(), "expected ',' after map key type")
	}

	p.Next() // skip ','

	valueType, err := ParseType(p)
	if err != nil {
		return nil, err
	}

	if p.Peek().Type != token.CloseAngle {
		return nil, NewError(p.Peek(), "expected '>' after map value type")
	}

	p.Next() // skip '>'

	return &ast.Map{
		Token: mapTok,
		Key:   keyType,
		Value: valueType,
	}, nil
}

func ParseMapKeyType(p *Parser) (ast.Type, error) {
	switch p.Peek().Type {
	case token.Int8, token.Int16, token.Int32, token.Int64:
		return ParseType(p)
	case token.Uint8, token.Uint16, token.Uint32, token.Uint64:
		return ParseType(p)
	case token.String:
		return ParseType(p)
	case token.Byte:
		return ParseType(p)
	default:
		return nil, NewError(p.Peek(), "expected map key type to be comparable")
	}
}

func ParseArrayType(p *Parser) (*ast.Array, error) {
	if p.Peek().Type != token.Array {
		return nil, NewError(p.Peek(), "expected 'array' keyword")
	}

	arrayTok := p.Next()

	arrayType, err := ParseType(p)
	if err != nil {
		return nil, err
	}

	return &ast.Array{
		Token: arrayTok,
		Type:  arrayType,
	}, nil
}

func extractTypeBits(prefix string, value string) int {
	// The resason why we don't return an error here is because
	// scanner already give us int8 ... float64 values and it has already
	// been validated.
	result, _ := strconv.ParseInt(value[len(prefix):], 10, 64)
	return int(result)
}

// Parse Service

func ParseService(p *Parser) (service *ast.Service, err error) {
	if p.Peek().Type != token.Service {
		return nil, NewError(p.Peek(), "expected service keyword")
	}

	service = &ast.Service{Token: p.Next()}

	if p.Peek().Type != token.Identifier {
		return nil, NewError(p.Peek(), "expected identifier for defining a service")
	}

	nameTok := p.Next()

	if !strcase.IsPascal(nameTok.Value) {
		return nil, NewError(nameTok, "service name must be in PascalCase format")
	}

	if strings.HasPrefix(nameTok.Value, "Http") {
		service.Type = ast.ServiceHTTP
	} else if strings.HasPrefix(nameTok.Value, "Rpc") {
		service.Type = ast.ServiceRPC
	} else {
		return nil, NewError(nameTok, "service name must start with 'Http' or 'Rpc'")
	}

	service.Name = &ast.Identifier{Token: nameTok}

	if p.Peek().Type != token.OpenCurly {
		return nil, NewError(p.Peek(), "expected '{' after service declaration")
	}

	if len(p.comments) > 0 {
		service.AddComments(p.comments...)
		p.comments = p.comments[:0]
	}

	p.Next() // skip '{'

	for {
		peek := p.Peek()

		if peek.Type == token.CloseCurly {
			break
		}

		if peek.Type == token.Comment {
			comment, err := ParseComment(p)
			if err != nil {
				return nil, err
			}

			p.comments = append(p.comments, comment)
			continue
		}

		method, err := ParseServiceMethod(p)
		if err != nil {
			return nil, err
		}

		service.Methods = append(service.Methods, method)
	}

	p.Next() // skip '}'

	return service, nil
}

func ParseServiceMethod(p *Parser) (method *ast.Method, err error) {
	method = &ast.Method{
		Args:    make([]*ast.Arg, 0),
		Returns: make([]*ast.Return, 0),
		Options: &ast.Options{
			List:     make([]*ast.Option, 0),
			Comments: make([]*ast.Comment, 0),
		},
		Comments: make([]*ast.Comment, 0),
	}

	if p.Peek().Type != token.Identifier {
		return nil, NewError(p.Peek(), "expected identifier for defining a service method")
	}

	nameTok := p.Next()

	if !strcase.IsPascal(nameTok.Value) {
		return nil, NewError(nameTok, "service method name must be in PascalCase format")
	}

	method.Name = &ast.Identifier{Token: nameTok}

	if p.Peek().Type != token.OpenParen {
		return nil, NewError(p.Peek(), "expected '(' after service method name")
	}

	p.Next() // skip '('

	for p.Peek().Type != token.CloseParen {
		arg, err := ParseServiceMethodArgument(p)
		if err != nil {
			return nil, err
		}

		method.Args = append(method.Args, arg)
	}

	p.Next() // skip ')'

	if p.Peek().Type == token.Return {
		p.Next() // skip =>

		if p.Peek().Type != token.OpenParen {
			return nil, NewError(p.Peek(), "expected '(' after '=>'")
		}

		p.Next() // skip '('

		for p.Peek().Type != token.CloseParen {
			ret, err := ParseServiceMethodReturnArg(p)
			if err != nil {
				return nil, err
			}

			method.Returns = append(method.Returns, ret)
		}

		p.Next() // skip ')'
	}

	if len(p.comments) > 0 {
		method.AddComments(p.comments...)
		p.comments = p.comments[:0]
	}

	// we return early if there are no options
	// as options are defined by curly braces
	if p.Peek().Type == token.OpenCurly {
		method.Options, err = ParseOptions(p)
		if err != nil {
			return nil, err
		}
	}

	return method, nil
}

func ParseServiceMethodArgument(p *Parser) (arg *ast.Arg, err error) {
	if p.Peek().Type != token.Identifier {
		return nil, NewError(p.Peek(), "expected identifier for defining a service method argument")
	}

	nameTok := p.Next()

	if !strcase.IsCamel(nameTok.Value) {
		return nil, NewError(nameTok, "service method argument name must be in camelCase format")
	}

	arg = &ast.Arg{Name: &ast.Identifier{Token: nameTok}}

	if p.Peek().Type != token.Colon {
		return nil, NewError(p.Peek(), "expected ':' after service method argument name")
	}

	p.Next() // skip ':'

	if p.Peek().Type == token.Stream {
		arg.Stream = true
		p.Next() // skip 'stream'
	}

	arg.Type, err = ParseType(p)
	if err != nil {
		return nil, err
	}

	if p.Peek().Type == token.Comma {
		p.Next() // skip ','
	}

	return arg, nil
}

func ParseServiceMethodReturnArg(p *Parser) (ret *ast.Return, err error) {
	if p.Peek().Type != token.Identifier {
		return nil, NewError(p.Peek(), "expected identifier for defining a service method argument")
	}

	nameTok := p.Next()

	if !strcase.IsCamel(nameTok.Value) {
		return nil, NewError(nameTok, "service method argument name must be in camelCase format")
	}

	ret = &ast.Return{Name: &ast.Identifier{Token: nameTok}}

	if p.Peek().Type != token.Colon {
		return nil, NewError(p.Peek(), "expected ':' after service method argument name")
	}

	p.Next() // skip ':'

	if p.Peek().Type == token.Stream {
		ret.Stream = true
		p.Next() // skip 'stream'
	}

	ret.Type, err = ParseType(p)
	if err != nil {
		return nil, err
	}

	if p.Peek().Type == token.Comma {
		p.Next() // skip ','
	}

	return ret, nil
}

// Parser Custom Error

func ParseCustomError(p *Parser) (customError *ast.CustomError, err error) {
	if p.Peek().Type != token.CustomError {
		return nil, NewError(p.Peek(), "expected 'error' keyword")
	}

	customError = &ast.CustomError{Token: p.Next()}

	if p.Peek().Type != token.Identifier {
		return nil, NewError(p.Peek(), "expected identifier for defining a custom error")
	}

	nameTok := p.Next()

	if !strcase.IsPascal(nameTok.Value) {
		return nil, NewError(nameTok, "custom error name must be in Pascal Case format")
	}

	customError.Name = &ast.Identifier{Token: nameTok}

	if p.Peek().Type != token.OpenCurly {
		return nil, NewError(p.Peek(), "expected '{' after custom error declaration")
	}

	p.Next() // skip '{'

	// parse Code, HttpStatus and Msg (3 times)
	for {
		peek := p.Peek()
		if peek.Type == token.CloseCurly {
			break
		}

		if peek.Type == token.Comment {
			comment, err := ParseComment(p)
			if err != nil {
				return nil, err
			}

			p.comments = append(p.comments, comment)
			continue
		}

		err = parseCustomErrorValues(p, customError)
		if err != nil {
			return nil, err
		}
	}

	p.Next() // skip '}'

	if customError.Msg == nil {
		return nil, NewError(customError.Token, "message is not defined in custom error")
	}

	if len(p.comments) > 0 {
		customError.AddComments(p.comments...)
		p.comments = p.comments[:0]
	}

	return customError, nil
}

func parseCustomErrorValues(p *Parser, customError *ast.CustomError) (err error) {
	if p.Peek().Type != token.Identifier {
		return NewError(p.Peek(), "expected identifier for defining a custom error value")
	}

	switch p.Peek().Value {
	case "Code":
		return parseCustomErrorCode(p, customError)
	case "Msg":
		return parseCustomErrorMsg(p, customError)
	}

	return NewError(p.Peek(), "unexpected field name in custom error")
}

func parseCustomErrorCode(p *Parser, customError *ast.CustomError) (err error) {
	if customError.Code != 0 {
		return NewError(p.Peek(), "code is already defined in custom error")
	}

	p.Next() // skip 'Code'

	if p.Peek().Type != token.Assign {
		return NewError(p.Peek(), "expected '=' after 'Code'")
	}

	p.Next() // skip '='

	if p.Peek().Type != token.ConstInt {
		return NewError(p.Peek(), "expected integer value for 'Code'")
	}

	codeValue, err := ParseValue(p)
	if err != nil {
		return err
	}

	customError.Code = codeValue.(*ast.ValueInt).Value

	return nil
}

func parseCustomErrorMsg(p *Parser, customError *ast.CustomError) (err error) {
	if customError.Msg != nil {
		return NewError(p.Peek(), "Msg is already defined in custom error")
	}

	p.Next() // skip 'Msg'

	if p.Peek().Type != token.Assign {
		return NewError(p.Peek(), "expected '=' after 'Msg'")
	}

	p.Next() // skip '='

	msgValue, err := ParseValue(p)
	if err != nil {
		return err
	}

	stringMsgValue, ok := msgValue.(*ast.ValueString)
	if !ok {
		return NewError(p.Peek(), "expected string value for 'Msg'")
	}

	customError.Msg = stringMsgValue

	return nil
}

// Parse Document

func ParseDocument(p *Parser) (*ast.Document, error) {
	doc := &ast.Document{}

	for p.Peek().Type != token.EOF {
		switch p.Peek().Type {
		case token.Comment:
			comment, err := ParseComment(p)
			if err != nil {
				return nil, err
			}

			p.comments = append(p.comments, comment)

		case token.Const:
			constant, err := ParseConst(p)
			if err != nil {
				return nil, err
			}

			doc.Consts = append(doc.Consts, constant)

			if len(p.comments) > 0 {
				constant.AddComments(p.comments...)
				p.comments = p.comments[:0]
			}

		case token.Enum:
			enum, err := ParseEnum(p)
			if err != nil {
				return nil, err
			}

			doc.Enums = append(doc.Enums, enum)

		case token.Model:
			model, err := ParseModel(p)
			if err != nil {
				return nil, err
			}

			doc.Models = append(doc.Models, model)

		case token.Service:
			service, err := ParseService(p)
			if err != nil {
				return nil, err
			}

			doc.Services = append(doc.Services, service)

		case token.CustomError:
			customError, err := ParseCustomError(p)
			if err != nil {
				return nil, err
			}

			doc.Errors = append(doc.Errors, customError)

		default:
			return nil, NewError(p.Peek(), "unexpected token")
		}
	}

	if len(p.comments) > 0 {
		doc.AddComments(p.comments...)
		p.comments = nil
	}

	return doc, nil
}

// Parse Value

func parseBytesNumber(value string) (number string, scale ast.ByteSize) {
	switch value[len(value)-2] {
	case 'k':
		scale = ast.ByteSizeKB
	case 'm':
		scale = ast.ByteSizeMB
	case 'g':
		scale = ast.ByteSizeGB
	case 't':
		scale = ast.ByteSizeTB
	case 'p':
		scale = ast.ByteSizePB
	case 'e':
		scale = ast.ByteSizeEB
	default:
		return value[:len(value)-1], 1
	}

	return value[:len(value)-2], scale
}

func parseDurationNumber(value string) (number string, scale ast.DurationScale) {
	switch value[len(value)-2] {
	case 'n':
		scale = ast.DurationScaleNanosecond
		return value[:len(value)-2], scale
	case 'u':
		scale = ast.DurationScaleMicrosecond
		return value[:len(value)-2], scale
	case 'm':
		scale = ast.DurationScaleMillisecond
		return value[:len(value)-2], scale
	default:
		switch value[len(value)-1] {
		case 's':
			scale = ast.DurationScaleSecond
		case 'm':
			scale = ast.DurationScaleMinute
		case 'h':
			scale = ast.DurationScaleHour
		}
		return value[:len(value)-1], scale
	}
}

func ParseValue(p *Parser) (value ast.Value, err error) {
	peekTok := p.Peek()

	switch peekTok.Type {
	case token.ConstBytes:
		num, scale := parseBytesNumber(strings.ReplaceAll(peekTok.Value, "_", ""))
		integer, err := strconv.ParseInt(num, 10, 64)
		if err != nil {
			return nil, NewError(peekTok, "failed to parse int value for bytes size: %s", err.Error())
		}
		value = &ast.ValueByteSize{
			Token: peekTok,
			Value: integer,
			Scale: scale,
		}
	case token.ConstDuration:
		num, scale := parseDurationNumber(strings.ReplaceAll(peekTok.Value, "_", ""))
		integer, err := strconv.ParseInt(num, 10, 64)
		if err != nil {
			return nil, NewError(peekTok, "failed to parse int value for duration size: %s", err)
		}
		value = &ast.ValueDuration{
			Token: peekTok,
			Value: integer,
			Scale: scale,
		}
	case token.ConstFloat:
		float, err := strconv.ParseFloat(strings.ReplaceAll(peekTok.Value, "_", ""), 64)
		if err != nil {
			return nil, NewError(peekTok, "failed to parse float value: %s", err)
		}
		value = &ast.ValueFloat{
			Token: peekTok,
			Value: float,
			Size:  getFloatSize(float),
		}
	case token.ConstInt:
		integer, err := strconv.ParseInt(strings.ReplaceAll(peekTok.Value, "_", ""), 10, 64)
		if err != nil {
			return nil, NewError(peekTok, "failed to parse int value: %s", err)
		}
		value = &ast.ValueInt{
			Token:   peekTok,
			Value:   integer,
			Defined: true,
			Size:    getIntSize(integer, integer),
		}
	case token.ConstBool:
		boolean, err := strconv.ParseBool(peekTok.Value)
		if err != nil {
			return nil, NewError(peekTok, "failed to parse bool value: %s", err)
		}
		value = &ast.ValueBool{
			Token:       peekTok,
			Value:       boolean,
			UserDefined: true,
		}
	case token.ConstNull:
		value = &ast.ValueNull{
			Token: peekTok,
		}
	case token.ConstStringSingleQuote, token.ConstStringDoubleQuote, token.ConstStringBacktickQoute:
		value = &ast.ValueString{
			Token: peekTok,
			Value: peekTok.Value,
		}
	case token.Identifier:
		value = &ast.ValueVariable{
			Token: peekTok,
		}
	default:
		return nil, NewError(peekTok, "expected one of the following, 'int', 'float', 'bool', 'null', 'string' values or identifier, got %s", peekTok.Type)
	}

	p.Next() // skip value if no error

	return value, nil
}

// find out about the min size for integer based on min and max values
// 8, –128, 127
// 16, –32768, 32767
// 32, -2147483648, 2147483647
// 64, -9223372036854775808, 9223372036854775807
func getIntSize(min, max int64) int {
	if min >= -128 && max <= 127 {
		return 8
	} else if min >= -32768 && max <= 32767 {
		return 16
	} else if min >= -2147483648 && max <= 2147483647 {
		return 32
	} else {
		return 64
	}
}

func getFloatSize(value float64) int {
	if value >= math.SmallestNonzeroFloat32 && value <= math.MaxFloat32 {
		return 32
	}
	return 64
}
