package main

import (
	"math"
	"strconv"
	"strings"

	"ella.to/ella/internal/strcase"
)

type Parser struct {
	tokens   TokenIterator
	nextTok  *Token
	currTok  *Token
	comments []*Comment
}

func (p *Parser) Current() *Token {
	return p.currTok
}

func (p *Parser) Next() *Token {
	if p.nextTok != nil {
		p.currTok = p.nextTok
		p.nextTok = nil
	} else {
		p.currTok = p.tokens.NextToken()
	}

	return p.currTok
}

func (p *Parser) Peek() *Token {
	if p.nextTok == nil {
		p.nextTok = p.tokens.NextToken()
	}

	return p.nextTok
}

func NewParser(input string) *Parser {
	tokenEmitter := NewEmitterIterator()
	go Start(tokenEmitter, Lex, input)
	return &Parser{
		tokens: tokenEmitter,
	}
}

func NewParserWithFilenames(filenames ...string) *Parser {
	tokenEmitter := NewEmitterIterator()
	go StartWithFilenames(tokenEmitter, Lex, filenames...)
	return &Parser{
		tokens: tokenEmitter,
	}
}

// Parse Comment

func ParseComment(p *Parser) (*Comment, error) {
	if p.Peek().Type != TokComment {
		return nil, NewError(p.Peek(), "expected comment but got %s", p.Peek().Type)
	}

	return &Comment{Token: p.Next()}, nil
}

// Parse Contsnant

func ParseConst(p *Parser) (*Const, error) {
	if p.Peek().Type != TokConst {
		return nil, NewError(p.Peek(), "expected const, got %s", p.Peek().Type)
	}

	constant := &Const{Token: p.Next()}

	if p.Peek().Type != TokIdentifier {
		return nil, NewError(p.Peek(), "expected identifier after const keyword, got %s", p.Peek().Type)
	}

	constant.Identifier = &Identifier{Token: p.Next()}

	if p.Peek().Type != TokAssign {
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

func ParseEnum(p *Parser) (enum *Enum, err error) {
	if p.Peek().Type != TokEnum {
		return nil, NewError(p.Peek(), "expected 'enum' keyword")
	}

	enum = &Enum{Token: p.Next()}

	if p.Peek().Type != TokIdentifier {
		return nil, NewError(p.Peek(), "expected identifier for defining an enum")
	}

	nameTok := p.Next()

	if !strcase.IsPascal(nameTok.Value) {
		return nil, NewError(nameTok, "enum name must be in Pascal Case format")
	}

	enum.Name = &Identifier{Token: nameTok}

	if p.Peek().Type != TokOpenCurly {
		return nil, NewError(p.Peek(), "expected '{' after enum declaration")
	}

	p.Next() // skip '{'

	for {
		peek := p.Peek()

		if peek.Type == TokCloseCurly {
			break
		}

		if peek.Type == TokIdentifier {
			set, err := parseEnumSet(p)
			if err != nil {
				return nil, err
			}

			set.AddComments(p.comments...)
			p.comments = p.comments[:0]

			enum.Sets = append(enum.Sets, set)
			continue
		}

		if peek.Type == TokComment {
			comment, err := ParseComment(p)
			if err != nil {
				return nil, err
			}

			comment.Position = CommentBottom
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

		set.Value = &ValueInt{
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

func parseEnumSet(p *Parser) (*EnumSet, error) {
	if p.Peek().Type != TokIdentifier {
		return nil, NewError(p.Peek(), "expected identifier for defining an enum constant")
	}

	nameTok := p.Next()

	if nameTok.Value != "_" && !strcase.IsPascal(nameTok.Value) {
		return nil, NewError(nameTok, "enum's set name must be in Pascal Case format")
	}

	if p.Peek().Type != TokAssign {
		return &EnumSet{
			Name: &Identifier{Token: nameTok},
			Value: &ValueInt{
				Value: 0,
			},
		}, nil
	}

	p.Next() // skip '='

	if p.Peek().Type != TokConstInt {
		return nil, NewError(p.Peek(), "expected constant integer value for defining an enum set value")
	}

	valueTok := p.Next()
	value, err := strconv.ParseInt(strings.ReplaceAll(valueTok.Value, "_", ""), 10, 64)
	if err != nil {
		return nil, NewError(valueTok, "invalid integer value for defining an enum constant value: %s", err)
	}

	return &EnumSet{
		Name: &Identifier{Token: nameTok},
		Value: &ValueInt{
			Token:   valueTok,
			Value:   value,
			Defined: true,
		},
		Defined: true,
	}, nil
}

// Parse Option

func ParseOption(p *Parser) (option *Option, err error) {
	if p.Peek().Type != TokIdentifier {
		return nil, NewError(p.Peek(), "expected identifier for defining a message field option")
	}

	nameTok := p.Next()

	option = &Option{
		Name: &Identifier{Token: nameTok},
	}

	if p.Peek().Type != TokAssign {
		option.Value = &ValueBool{
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

func ParseOptions(p *Parser) (*Options, error) {
	options := &Options{
		List:     make([]*Option, 0),
		Comments: make([]*Comment, 0),
	}

	p.Next() // skip '{'

	for {
		peek := p.Peek()

		if peek.Type == TokCloseCurly {
			break
		}

		if peek.Type == TokComment {
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
			comment.Position = CommentBottom
		}

		options.AddComments(p.comments...)
		p.comments = p.comments[:0]
	}

	return options, nil
}

// Parse Model

func ParseModel(p *Parser) (*Model, error) {
	if p.Peek().Type != TokModel {
		return nil, NewError(p.Peek(), "expected 'model' keyword")
	}

	model := &Model{Token: p.Next()}

	if p.Peek().Type != TokIdentifier {
		return nil, NewError(p.Peek(), "expected identifier for defining a model")
	}

	nameTok := p.Next()

	if !strcase.IsPascal(nameTok.Value) {
		return nil, NewError(nameTok, "model name must be in PascalCase format")
	}

	model.Name = &Identifier{Token: nameTok}

	if p.Peek().Type != TokOpenCurly {
		return nil, NewError(p.Peek(), "expected '{' after model declaration")
	}

	p.Next() // skip '{'

	if len(p.comments) > 0 {
		model.AddComments(p.comments...)
		p.comments = p.comments[:0]
	}

	for {
		peek := p.Peek()

		if peek.Type == TokCloseCurly {
			break
		}

		if peek.Type == TokComment {
			comment, err := ParseComment(p)
			if err != nil {
				return nil, err
			}

			p.comments = append(p.comments, comment)
			continue
		}

		if peek.Type == TokExtend {
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
			comment.Position = CommentBottom
		}

		model.AddComments(p.comments...)
		p.comments = p.comments[:0]
	}

	return model, nil
}

func ParseExtend(p *Parser) (*Extend, error) {
	if p.Peek().Type != TokExtend {
		return nil, NewError(p.Peek(), "expected '...' keyword")
	}

	p.Next() // skip '...'

	if p.Peek().Type != TokIdentifier {
		return nil, NewError(p.Peek(), "expected identifier for extending a message")
	}

	nameTok := p.Next()

	if !strcase.IsPascal(nameTok.Value) {
		return nil, NewError(nameTok, "extend message name must be in PascalCase format")
	}

	return &Extend{
		Name:     &Identifier{Token: nameTok},
		Comments: make([]*Comment, 0),
	}, nil
}

func ParseModelField(p *Parser) (field *Field, err error) {
	if p.Peek().Type != TokIdentifier {
		return nil, NewError(p.Peek(), "expected identifier for defining a message field")
	}

	nameTok := p.Next()

	if !strcase.IsPascal(nameTok.Value) {
		return nil, NewError(nameTok, "message field name must be in PascalCase format")
	}

	field = &Field{
		Name:     &Identifier{Token: nameTok},
		Options:  &Options{List: make([]*Option, 0)},
		Comments: make([]*Comment, 0),
	}

	peek := p.Peek()

	switch peek.Type {
	case TokOptional:
		field.IsOptional = true
		p.Next() // skip '?'

		if p.Peek().Type != TokColon {
			return nil, NewError(p.Peek(), "expected ':' after '?'")
		}
		p.Next() // skip ':'

	case TokColon:
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

	if p.Peek().Type != TokOpenCurly {
		return field, nil
	}

	field.Options, err = ParseOptions(p)
	if err != nil {
		return nil, err
	}

	return field, nil
}

// Parse Type

func ParseType(p *Parser) (Type, error) {
	peek := p.Peek()

	switch peek.Type {
	case TokMap:
		return ParseMapType(p)
	case TokArray:
		return ParseArrayType(p)
	case TokBool:
		return &Bool{Token: p.Next()}, nil
	case TokByte:
		return &Byte{Token: p.Next()}, nil
	case TokInt8, TokInt16, TokInt32, TokInt64:
		tok := p.Next()
		return &Int{
			Token: tok,
			Size:  extractTypeBits("int", tok.Value),
		}, nil
	case TokUint8, TokUint16, TokUint32, TokUint64:
		tok := p.Next()
		return &Uint{
			Token: tok,
			Size:  extractTypeBits("uint", tok.Value),
		}, nil
	case TokFloat32, TokFloat64:
		tok := p.Next()
		return &Float{
			Token: tok,
			Size:  extractTypeBits("float", tok.Value),
		}, nil
	case TokTimestamp:
		return &Timestamp{Token: p.Next()}, nil
	case TokString:
		return &String{Token: p.Next()}, nil
	case TokAny:
		return &Any{Token: p.Next()}, nil
	case TokIdentifier:
		nameTok := p.Next()

		if !strcase.IsPascal(nameTok.Value) {
			return nil, NewError(nameTok, "custom type name must be in PascalCase format")
		}

		return &CustomType{Token: nameTok}, nil
	default:
		return nil, NewError(peek, "expected type")
	}
}

func ParseMapType(p *Parser) (*Map, error) {
	if p.Peek().Type != TokMap {
		return nil, NewError(p.Peek(), "expected 'map' keyword")
	}

	mapTok := p.Next()

	if p.Peek().Type != TokOpenAngle {
		return nil, NewError(p.Peek(), "expected '<' after 'map' keyword")
	}

	p.Next() // skip '<'

	keyType, err := ParseMapKeyType(p)
	if err != nil {
		return nil, err
	}

	if p.Peek().Type != TokComma {
		return nil, NewError(p.Peek(), "expected ',' after map key type")
	}

	p.Next() // skip ','

	valueType, err := ParseType(p)
	if err != nil {
		return nil, err
	}

	if p.Peek().Type != TokCloseAngle {
		return nil, NewError(p.Peek(), "expected '>' after map value type")
	}

	p.Next() // skip '>'

	return &Map{
		Token: mapTok,
		Key:   keyType,
		Value: valueType,
	}, nil
}

func ParseMapKeyType(p *Parser) (Type, error) {
	switch p.Peek().Type {
	case TokInt8, TokInt16, TokInt32, TokInt64:
		return ParseType(p)
	case TokUint8, TokUint16, TokUint32, TokUint64:
		return ParseType(p)
	case TokString:
		return ParseType(p)
	case TokByte:
		return ParseType(p)
	default:
		return nil, NewError(p.Peek(), "expected map key type to be comparable")
	}
}

func ParseArrayType(p *Parser) (*Array, error) {
	if p.Peek().Type != TokArray {
		return nil, NewError(p.Peek(), "expected 'array' keyword")
	}

	arrayTok := p.Next()

	arrayType, err := ParseType(p)
	if err != nil {
		return nil, err
	}

	return &Array{
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

func ParseService(p *Parser) (service *Service, err error) {
	if p.Peek().Type != TokService {
		return nil, NewError(p.Peek(), "expected service keyword")
	}

	service = &Service{Token: p.Next()}

	if p.Peek().Type != TokIdentifier {
		return nil, NewError(p.Peek(), "expected identifier for defining a service")
	}

	nameTok := p.Next()

	if !strcase.IsPascal(nameTok.Value) {
		return nil, NewError(nameTok, "service name must be in PascalCase format")
	}

	if strings.HasPrefix(nameTok.Value, "Http") {
		service.Type = ServiceHTTP
	} else if strings.HasPrefix(nameTok.Value, "Rpc") {
		service.Type = ServiceRPC
	} else {
		return nil, NewError(nameTok, "service name must start with 'Http' or 'Rpc'")
	}

	service.Name = &Identifier{Token: nameTok}

	if p.Peek().Type != TokOpenCurly {
		return nil, NewError(p.Peek(), "expected '{' after service declaration")
	}

	if len(p.comments) > 0 {
		service.AddComments(p.comments...)
		p.comments = p.comments[:0]
	}

	p.Next() // skip '{'

	for {
		peek := p.Peek()

		if peek.Type == TokCloseCurly {
			break
		}

		if peek.Type == TokComment {
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

func ParseServiceMethod(p *Parser) (method *Method, err error) {
	method = &Method{
		Args:    make([]*Arg, 0),
		Returns: make([]*Return, 0),
		Options: &Options{
			List:     make([]*Option, 0),
			Comments: make([]*Comment, 0),
		},
		Comments: make([]*Comment, 0),
	}

	if p.Peek().Type != TokIdentifier {
		return nil, NewError(p.Peek(), "expected identifier for defining a service method")
	}

	nameTok := p.Next()

	if !strcase.IsPascal(nameTok.Value) {
		return nil, NewError(nameTok, "service method name must be in PascalCase format")
	}

	method.Name = &Identifier{Token: nameTok}

	if p.Peek().Type != TokOpenParen {
		return nil, NewError(p.Peek(), "expected '(' after service method name")
	}

	p.Next() // skip '('

	for p.Peek().Type != TokCloseParen {
		arg, err := ParseServiceMethodArgument(p)
		if err != nil {
			return nil, err
		}

		method.Args = append(method.Args, arg)
	}

	p.Next() // skip ')'

	if p.Peek().Type == TokReturn {
		p.Next() // skip =>

		if p.Peek().Type != TokOpenParen {
			return nil, NewError(p.Peek(), "expected '(' after '=>'")
		}

		p.Next() // skip '('

		for p.Peek().Type != TokCloseParen {
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
	if p.Peek().Type == TokOpenCurly {
		method.Options, err = ParseOptions(p)
		if err != nil {
			return nil, err
		}
	}

	return method, nil
}

func ParseServiceMethodArgument(p *Parser) (arg *Arg, err error) {
	if p.Peek().Type != TokIdentifier {
		return nil, NewError(p.Peek(), "expected identifier for defining a service method argument")
	}

	nameTok := p.Next()

	if !strcase.IsCamel(nameTok.Value) {
		return nil, NewError(nameTok, "service method argument name must be in camelCase format")
	}

	arg = &Arg{Name: &Identifier{Token: nameTok}}

	if p.Peek().Type != TokColon {
		return nil, NewError(p.Peek(), "expected ':' after service method argument name")
	}

	p.Next() // skip ':'

	if p.Peek().Type == TokStream {
		arg.Stream = true
		p.Next() // skip 'stream'
	}

	arg.Type, err = ParseType(p)
	if err != nil {
		return nil, err
	}

	if p.Peek().Type == TokComma {
		p.Next() // skip ','
	}

	return arg, nil
}

func ParseServiceMethodReturnArg(p *Parser) (ret *Return, err error) {
	if p.Peek().Type != TokIdentifier {
		return nil, NewError(p.Peek(), "expected identifier for defining a service method argument")
	}

	nameTok := p.Next()

	if !strcase.IsCamel(nameTok.Value) {
		return nil, NewError(nameTok, "service method argument name must be in camelCase format")
	}

	ret = &Return{Name: &Identifier{Token: nameTok}}

	if p.Peek().Type != TokColon {
		return nil, NewError(p.Peek(), "expected ':' after service method argument name")
	}

	p.Next() // skip ':'

	if p.Peek().Type == TokStream {
		ret.Stream = true
		p.Next() // skip 'stream'
	}

	ret.Type, err = ParseType(p)
	if err != nil {
		return nil, err
	}

	if p.Peek().Type == TokComma {
		p.Next() // skip ','
	}

	return ret, nil
}

// Parser Custom Error

func ParseCustomError(p *Parser) (customError *CustomError, err error) {
	if p.Peek().Type != TokCustomError {
		return nil, NewError(p.Peek(), "expected 'error' keyword")
	}

	customError = &CustomError{Token: p.Next()}

	if p.Peek().Type != TokIdentifier {
		return nil, NewError(p.Peek(), "expected identifier for defining a custom error")
	}

	nameTok := p.Next()

	if !strcase.IsPascal(nameTok.Value) {
		return nil, NewError(nameTok, "custom error name must be in Pascal Case format")
	}

	customError.Name = &Identifier{Token: nameTok}

	if p.Peek().Type != TokOpenCurly {
		return nil, NewError(p.Peek(), "expected '{' after custom error declaration")
	}

	p.Next() // skip '{'

	// parse Code, HttpStatus and Msg (3 times)
	for {
		peek := p.Peek()
		if peek.Type == TokCloseCurly {
			break
		}

		if peek.Type == TokComment {
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

func parseCustomErrorValues(p *Parser, customError *CustomError) (err error) {
	if p.Peek().Type != TokIdentifier {
		return NewError(p.Peek(), "expected identifier for defining a custom error value")
	}

	switch p.Peek().Value {
	case "Code":
		return parseCustomErrorCode(p, customError)
	case "HttpStatus":
		return parseCustomErrorHttpStatus(p, customError)
	case "Msg":
		return parseCustomErrorMsg(p, customError)
	}

	return NewError(p.Peek(), "unexpected field name in custom error")
}

func parseCustomErrorCode(p *Parser, customError *CustomError) (err error) {
	if customError.Code != 0 {
		return NewError(p.Peek(), "code is already defined in custom error")
	}

	p.Next() // skip 'Code'

	if p.Peek().Type != TokAssign {
		return NewError(p.Peek(), "expected '=' after 'Code'")
	}

	p.Next() // skip '='

	if p.Peek().Type != TokConstInt {
		return NewError(p.Peek(), "expected integer value for 'Code'")
	}

	codeValue, err := ParseValue(p)
	if err != nil {
		return err
	}

	customError.Code = codeValue.(*ValueInt).Value

	return nil
}

func parseCustomErrorHttpStatus(p *Parser, customError *CustomError) (err error) {
	if customError.HttpStatus != 0 {
		return NewError(p.Peek(), "HttpStatus is already defined in custom error")
	}

	p.Next() // skip 'HttpStatus'

	if p.Peek().Type != TokAssign {
		return NewError(p.Peek(), "expected '=' after 'HttpStatus'")
	}

	p.Next() // skip '='

	if p.Peek().Type != TokIdentifier {
		return NewError(p.Peek(), "expected a HttpStatus value as a string value, e.g. NotFound")
	}

	httpStatus, ok := HttpStatusString2Code[p.Peek().Value]
	if !ok {
		return NewError(p.Peek(), "unexpected http status value")
	}

	customError.HttpStatus = httpStatus

	p.Next() // skip http status

	return nil
}

func parseCustomErrorMsg(p *Parser, customError *CustomError) (err error) {
	if customError.Msg != nil {
		return NewError(p.Peek(), "Msg is already defined in custom error")
	}

	p.Next() // skip 'Msg'

	if p.Peek().Type != TokAssign {
		return NewError(p.Peek(), "expected '=' after 'Msg'")
	}

	p.Next() // skip '='

	msgValue, err := ParseValue(p)
	if err != nil {
		return err
	}

	stringMsgValue, ok := msgValue.(*ValueString)
	if !ok {
		return NewError(p.Peek(), "expected string value for 'Msg'")
	}

	customError.Msg = stringMsgValue

	return nil
}

// Parse Document

func ParseDocument(p *Parser) (*Document, error) {
	doc := &Document{}

	for p.Peek().Type != TokEOF {
		switch p.Peek().Type {
		case TokComment:
			comment, err := ParseComment(p)
			if err != nil {
				return nil, err
			}

			p.comments = append(p.comments, comment)

		case TokConst:
			constant, err := ParseConst(p)
			if err != nil {
				return nil, err
			}

			doc.Consts = append(doc.Consts, constant)

			if len(p.comments) > 0 {
				constant.AddComments(p.comments...)
				p.comments = p.comments[:0]
			}

		case TokEnum:
			enum, err := ParseEnum(p)
			if err != nil {
				return nil, err
			}

			doc.Enums = append(doc.Enums, enum)

		case TokModel:
			model, err := ParseModel(p)
			if err != nil {
				return nil, err
			}

			doc.Models = append(doc.Models, model)

		case TokService:
			service, err := ParseService(p)
			if err != nil {
				return nil, err
			}

			doc.Services = append(doc.Services, service)

		case TokCustomError:
			customError, err := ParseCustomError(p)
			if err != nil {
				return nil, err
			}

			doc.Errors = append(doc.Errors, customError)
		}
	}

	if len(p.comments) > 0 {
		doc.AddComments(p.comments...)
		p.comments = nil
	}

	return doc, nil
}

// Parse Value

func parseBytesNumber(value string) (number string, scale ByteSize) {
	switch value[len(value)-2] {
	case 'k':
		scale = ByteSizeKB
	case 'm':
		scale = ByteSizeMB
	case 'g':
		scale = ByteSizeGB
	case 't':
		scale = ByteSizeTB
	case 'p':
		scale = ByteSizePB
	case 'e':
		scale = ByteSizeEB
	default:
		return value[:len(value)-1], 1
	}

	return value[:len(value)-2], scale
}

func parseDurationNumber(value string) (number string, scale DurationScale) {
	switch value[len(value)-2] {
	case 'n':
		scale = DurationScaleNanosecond
		return value[:len(value)-2], scale
	case 'u':
		scale = DurationScaleMicrosecond
		return value[:len(value)-2], scale
	case 'm':
		scale = DurationScaleMillisecond
		return value[:len(value)-2], scale
	default:
		switch value[len(value)-1] {
		case 's':
			scale = DurationScaleSecond
		case 'm':
			scale = DurationScaleMinute
		case 'h':
			scale = DurationScaleHour
		}
		return value[:len(value)-1], scale
	}
}

func ParseValue(p *Parser) (value Value, err error) {
	peekTok := p.Peek()

	switch peekTok.Type {
	case TokConstBytes:
		num, scale := parseBytesNumber(strings.ReplaceAll(peekTok.Value, "_", ""))
		integer, err := strconv.ParseInt(num, 10, 64)
		if err != nil {
			return nil, NewError(peekTok, "failed to parse int value for bytes size: %s", err.Error())
		}
		value = &ValueByteSize{
			Token: peekTok,
			Value: integer,
			Scale: scale,
		}
	case TokConstDuration:
		num, scale := parseDurationNumber(strings.ReplaceAll(peekTok.Value, "_", ""))
		integer, err := strconv.ParseInt(num, 10, 64)
		if err != nil {
			return nil, NewError(peekTok, "failed to parse int value for duration size: %s", err)
		}
		value = &ValueDuration{
			Token: peekTok,
			Value: integer,
			Scale: scale,
		}
	case TokConstFloat:
		float, err := strconv.ParseFloat(strings.ReplaceAll(peekTok.Value, "_", ""), 64)
		if err != nil {
			return nil, NewError(peekTok, "failed to parse float value: %s", err)
		}
		value = &ValueFloat{
			Token: peekTok,
			Value: float,
			Size:  getFloatSize(float),
		}
	case TokConstInt:
		integer, err := strconv.ParseInt(strings.ReplaceAll(peekTok.Value, "_", ""), 10, 64)
		if err != nil {
			return nil, NewError(peekTok, "failed to parse int value: %s", err)
		}
		value = &ValueInt{
			Token:   peekTok,
			Value:   integer,
			Defined: true,
			Size:    getIntSize(integer, integer),
		}
	case TokConstBool:
		boolean, err := strconv.ParseBool(peekTok.Value)
		if err != nil {
			return nil, NewError(peekTok, "failed to parse bool value: %s", err)
		}
		value = &ValueBool{
			Token:       peekTok,
			Value:       boolean,
			UserDefined: true,
		}
	case TokConstNull:
		value = &ValueNull{
			Token: peekTok,
		}
	case TokConstStringSingleQuote, TokConstStringDoubleQuote, TokConstStringBacktickQoute:
		value = &ValueString{
			Token: peekTok,
			Value: peekTok.Value,
		}
	case TokIdentifier:
		value = &ValueVariable{
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
