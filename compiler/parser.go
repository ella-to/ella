package compiler

type Parser struct {
	scanner   *Scanner
	nextToken *Token
	comments  []*Token
	scanErr   error
}

func NewParser(s *Scanner) *Parser {
	return &Parser{
		scanner:   s,
		nextToken: nil,
		comments:  make([]*Token, 0),
	}
}

func (p *Parser) scan() (*Token, error) {
	for {
		tok, err := p.scanner.Scan()
		if err != nil {
			return nil, err
		}
		if tok.Type == COMMENT {
			p.comments = append(p.comments, tok)
			continue
		}
		return tok, nil
	}
}

func (p *Parser) next() (*Token, error) {
	if p.nextToken != nil {
		tok := p.nextToken
		err := p.scanErr
		p.nextToken = nil
		p.scanErr = nil
		return tok, err
	}
	return p.scan()
}

func (p *Parser) peek() (*Token, error) {
	if p.nextToken == nil {
		var err error
		p.nextToken, err = p.scan()
		p.scanErr = err
	}
	return p.nextToken, p.scanErr
}

func (p *Parser) Parse() (*Program, error) {
	var nodes []Node
	var node Node

	for {
		tok, err := p.peek()
		if err != nil {
			return nil, err
		}
		if tok.Type == EOF {
			break
		}

		switch tok.Type {
		case CONST:
			node, err = p.parseConstDecl()
		case ENUM:
			node, err = p.parseEnumDecl()
		case MODEL:
			node, err = p.parseModelDecl()
		case SERVICE:
			node, err = p.parseServiceDecl()
		case CUSTOM_ERROR:
			node, err = p.parseErrorDecl()
		case ERROR:
			return nil, NewError(tok, tok.Lit)
		default:
			return nil, NewError(tok, "unexpected token: %s", tok.Type.String())
		}

		if err != nil {
			return nil, err
		}

		nodes = append(nodes, node)
	}

	return &Program{
		Nodes:    nodes,
		Comments: p.comments,
	}, nil
}

func (p *Parser) parseIdenExpr() (*IdenExpr, error) {
	idenTok, err := p.next()
	if err != nil {
		return nil, err
	}
	if idenTok.Type != IDENTIFIER {
		return nil, NewError(idenTok, "expected identifier, got %s", idenTok.Type.String())
	}

	return &IdenExpr{
		Token: idenTok,
		Name:  idenTok.Lit,
	}, nil
}

func (p *Parser) parseValueExpr() (Expr, error) {
	var err error

	valueTok, err := p.next()
	if err != nil {
		return nil, err
	}

	switch valueTok.Type {
	case CONST_NUMBER:
		expr := &ValueExprNumber{
			Token: valueTok,
			Type:  nil,
		}

		extra, err := p.peek()
		if err != nil {
			return nil, err
		}

		if extra.Type != IDENTIFIER {
			return expr, nil
		}

		switch extra.Lit {
		case "kb", "mb", "gb", "tb", "pb", "eb", "ms", "s", "m", "h":
			expr.Type, err = p.parseIdenExpr()
			if err != nil {
				return nil, err
			}
		}

		return expr, nil
	case CONST_STRING_SINGLE_QUOTE, CONST_STRING_DOUBLE_QUOTE, CONST_STRING_BACKTICK_QOUTE:
		return &ValueExprString{
			Token: valueTok,
		}, nil
	case CONST_BOOL:
		return &ValueExprBool{
			Token: valueTok,
		}, nil
	case CONST_NULL:
		return &ValueExprNull{
			Token: valueTok,
		}, nil
	case IDENTIFIER:
		// Check if identifier is followed by DOT (e.g., jetdrive.device.created)
		// This is not valid - user probably forgot to quote a string
		nextTok, err := p.peek()
		if err != nil {
			return nil, err
		}
		if nextTok.Type == DOT {
			return nil, NewError(valueTok, "unexpected identifier '%s' followed by '.'; did you mean to use a string like \"%s...\"?", valueTok.Lit, valueTok.Lit)
		}
		return &IdenExpr{
			Token: valueTok,
			Name:  valueTok.Lit,
		}, nil
	default:
		return nil, NewError(valueTok, "expected value, got %s", valueTok.Type.String())
	}
}

func (p *Parser) parseAssignmentStmt(withEqualOnly bool) (*AssignmentStmt, error) {
	var err error

	assignmentExpr := &AssignmentStmt{}

	assignmentExpr.Name, err = p.parseIdenExpr()
	if err != nil {
		return nil, err
	}

	tok, err := p.peek()
	if err != nil {
		return nil, err
	}

	var value Expr

	if tok.Type == EQUAL {
		// consume '='
		_, err = p.next()
		if err != nil {
			return nil, err
		}

		value, err = p.parseValueExpr()
		if err != nil {
			return nil, err
		}
	} else if withEqualOnly {
		return nil, NewError(tok, "expected '=' in assignment statement, got %s", tok.Type.String())
	} else {
		value = &ValueExprBool{
			Token: newInjectedToken(CONST_BOOL, "true"),
		}
	}

	assignmentExpr.Value = value

	return assignmentExpr, nil
}

func (p *Parser) parseConstDecl() (*ConstDecl, error) {
	var err error

	constDecl := &ConstDecl{}

	// consume 'const'
	constDecl.Token, err = p.next()
	if err != nil {
		return nil, err
	}

	constDecl.Assignment, err = p.parseAssignmentStmt(true)
	if err != nil {
		return nil, err
	}

	return constDecl, nil
}

func (p *Parser) parseEnumValue() (*DeclEnumSet, error) {
	var err error

	enumSet := &DeclEnumSet{}

	enumSet.Name, err = p.parseIdenExpr()
	if err != nil {
		return nil, err
	}

	peek, err := p.peek()
	if err != nil {
		return nil, err
	}

	if peek.Type != EQUAL {
		enumSet.IsDefined = false
		return enumSet, nil
	}

	// consume '='
	_, err = p.next()
	if err != nil {
		return nil, err
	}

	enumSet.Value, err = p.parseValueExpr()
	if err != nil {
		return nil, err
	}

	enumSet.IsDefined = true

	return enumSet, nil
}

func (p *Parser) parseEnumDecl() (*DeclEnum, error) {
	enumDecl := &DeclEnum{}

	// consume 'enum'
	var err error
	enumDecl.Token, err = p.next()
	if err != nil {
		return nil, err
	}

	name, err := p.parseIdenExpr()
	if err != nil {
		return nil, err
	}

	enumDecl.Name = name

	openCurlyTok, err := p.next()
	if err != nil {
		return nil, err
	}
	if openCurlyTok.Type != OPEN_CURLY {
		return nil, NewError(openCurlyTok, "expected '{' after identifier in enum declaration, got %s", openCurlyTok.Type.String())
	}

	for {
		peek, err := p.peek()
		if err != nil {
			return nil, err
		}
		if peek.Type == CLOSE_CURLY {
			break
		}

		enumSet, err := p.parseEnumValue()
		if err != nil {
			return nil, err
		}

		enumDecl.Values = append(enumDecl.Values, enumSet)
	}

	closeCurlyTok, err := p.next()
	if err != nil {
		return nil, err
	}
	if closeCurlyTok.Type != CLOSE_CURLY {
		return nil, NewError(closeCurlyTok, "expected '}' at the end of enum declaration, got %s", closeCurlyTok.Type.String())
	}

	enumDecl.CloseCurly = closeCurlyTok

	return enumDecl, nil
}

func (p *Parser) parseDeclType() (DeclType, error) {
	var err error

	tok, err := p.next()
	if err != nil {
		return nil, err
	}

	switch tok.Type {
	case INT8, INT16, INT32, INT64, UINT8, UINT16, UINT32, UINT64, FLOAT32, FLOAT64:
		return &DeclNumberType{
			Name: &IdenExpr{
				Token: tok,
				Name:  tok.Lit,
			},
		}, nil
	case STRING:
		return &DeclStringType{
			Name: &IdenExpr{
				Token: tok,
				Name:  tok.Lit,
			},
		}, nil
	case BYTE:
		return &DeclByteType{
			Name: &IdenExpr{
				Token: tok,
				Name:  tok.Lit,
			},
		}, nil
	case TIMESTAMP:
		return &DeclTimestampType{
			Name: &IdenExpr{
				Token: tok,
				Name:  tok.Lit,
			},
		}, nil
	case ANY:
		return &DeclAnyType{
			Name: &IdenExpr{
				Token: tok,
				Name:  tok.Lit,
			},
		}, nil
	case BOOL:
		return &DeclBoolType{
			Name: &IdenExpr{
				Token: tok,
				Name:  tok.Lit,
			},
		}, nil
	case MAP:
		mapType := &DeclMapType{
			Token: tok,
		}

		// consume '<'
		angleOpenTok, err := p.next()
		if err != nil {
			return nil, err
		}
		if angleOpenTok.Type != OPEN_ANGLE {
			return nil, NewError(angleOpenTok, "expected '<' after 'map' in map type declaration, got %s", angleOpenTok.Type.String())
		}

		mapType.KeyType, err = p.parseDeclType()
		if err != nil {
			return nil, err
		}

		commaTok, err := p.next()
		if err != nil {
			return nil, err
		}
		if commaTok.Type != COMMA {
			return nil, NewError(commaTok, "expected ',' after key type in map type declaration, got %s", commaTok.Type.String())
		}

		mapType.ValueType, err = p.parseDeclType()
		if err != nil {
			return nil, err
		}

		// consume '>'
		angleCloseTok, err := p.next()
		if err != nil {
			return nil, err
		}

		if angleCloseTok.Type != CLOSE_ANGLE {
			return nil, NewError(angleCloseTok, "expected '>' at the end of map type declaration, got %s", angleCloseTok.Type.String())
		}

		return mapType, nil
	case OPEN_SQURE:
		arrayType := &DeclArrayType{
			Token: tok,
		}

		peekTok, err := p.next()
		if err != nil {
			return nil, err
		}
		if peekTok.Type != CLOSE_SQURE {
			return nil, NewError(peekTok, "expected ']' after '[' in array type declaration, got %s", peekTok.Type.String())
		}

		arrayType.Type, err = p.parseDeclType()
		if err != nil {
			return nil, err
		}

		return arrayType, nil
	case IDENTIFIER:
		return &DeclCustomType{
			Name: &IdenExpr{
				Token: tok,
				Name:  tok.Lit,
			},
		}, nil
	default:
		return nil, NewError(tok, "expected type declaration, got %s", tok.Type.String())
	}
}

func (p *Parser) parseDeclModelField() (*DeclModelField, error) {
	var err error

	declModelField := &DeclModelField{}

	declModelField.Name, err = p.parseIdenExpr()
	if err != nil {
		return nil, err
	}

	colonTok, err := p.next()
	if err != nil {
		return nil, err
	}
	if colonTok.Type == OPTIONAL {
		declModelField.Optional = true
		colonTok, err = p.next()
		if err != nil {
			return nil, err
		}
	}
	if colonTok.Type != COLON {
		return nil, NewError(colonTok, "expected ':' after identifier in model field declaration, got %s", colonTok.Type.String())
	}

	declModelField.Type, err = p.parseDeclType()
	if err != nil {
		return nil, err
	}

	peek, err := p.peek()
	if err != nil {
		return nil, err
	}
	if peek.Type != OPEN_CURLY {
		return declModelField, nil
	}

	// consume '{'
	_, err = p.next()
	if err != nil {
		return nil, err
	}

	for {
		peek, err := p.peek()
		if err != nil {
			return nil, err
		}
		if peek.Type == CLOSE_CURLY {
			break
		}

		opt, err := p.parseAssignmentStmt(false)
		if err != nil {
			return nil, err
		}

		declModelField.Options = append(declModelField.Options, opt)
	}

	// consume '}'
	_, err = p.next()
	if err != nil {
		return nil, err
	}

	return declModelField, nil
}

func (p *Parser) parseModelDecl() (*DeclModel, error) {
	var err error

	modelDecl := &DeclModel{}

	// consume 'model'
	modelDecl.Token, err = p.next()
	if err != nil {
		return nil, err
	}

	modelDecl.Name, err = p.parseIdenExpr()
	if err != nil {
		return nil, err
	}

	tok, err := p.next()
	if err != nil {
		return nil, err
	}
	if tok.Type != OPEN_CURLY {
		return nil, NewError(tok, "expected '{' after identifier in model declaration, got %s", tok.Type.String())
	}

	for {
		peek, err := p.peek()
		if err != nil {
			return nil, err
		}
		if peek.Type == CLOSE_CURLY {
			break
		}

		if peek.Type == DOT {
			extend, err := p.parseExtendModelDecl()
			if err != nil {
				return nil, err
			}

			modelDecl.Extends = append(modelDecl.Extends, extend)
			continue
		}

		field, err := p.parseDeclModelField()
		if err != nil {
			return nil, err
		}

		modelDecl.Fields = append(modelDecl.Fields, field)
	}

	modelDecl.CloseCurly, err = p.next() // consume '}'
	if err != nil {
		return nil, err
	}

	return modelDecl, nil
}

func (p *Parser) parseExtendModelDecl() (*IdenExpr, error) {
	for range 3 {
		dotTok, err := p.next()
		if err != nil {
			return nil, err
		}
		if dotTok.Type != DOT {
			return nil, NewError(dotTok, "expected '.' in extend model declaration, got %s", dotTok.Type.String())
		}
	}

	extendModelName, err := p.parseIdenExpr()
	if err != nil {
		return nil, err
	}

	return extendModelName, nil
}

func (p *Parser) parseDeclNameTypePair() (*DeclNameTypePair, error) {
	var err error

	nameTypePair := &DeclNameTypePair{}

	nameTypePair.Name, err = p.parseIdenExpr()
	if err != nil {
		return nil, err
	}

	colonTok, err := p.next()
	if err != nil {
		return nil, err
	}
	if colonTok.Type != COLON {
		return nil, NewError(colonTok, "expected ':' after identifier in name-type pair declaration, got %s", colonTok.Type.String())
	}

	nameTypePair.Type, err = p.parseDeclType()
	if err != nil {
		return nil, err
	}

	return nameTypePair, nil
}

func (p *Parser) parseDeclServiceMethod() (*DeclServiceMethod, error) {
	var err error

	method := &DeclServiceMethod{}

	method.Name, err = p.parseIdenExpr()
	if err != nil {
		return nil, err
	}

	openParenTok, err := p.next()
	if err != nil {
		return nil, err
	}
	if openParenTok.Type != OPEN_PAREN {
		return nil, NewError(openParenTok, "expected '(' after identifier in service method declaration, got %s", openParenTok.Type.String())
	}

	for {
		peek, err := p.peek()
		if err != nil {
			return nil, err
		}
		if peek.Type == CLOSE_PAREN {
			break
		}

		arg, err := p.parseDeclNameTypePair()
		if err != nil {
			return nil, err
		}

		method.Args = append(method.Args, arg)

		peek, err = p.peek()
		if err != nil {
			return nil, err
		}
		if peek.Type == COMMA {
			_, _ = p.next() // consume ','
		}
	}

	closeParenTok, err := p.next()
	if err != nil {
		return nil, err
	}
	if closeParenTok.Type != CLOSE_PAREN {
		return nil, NewError(closeParenTok, "expected ')' at the end of service method arguments, got %s", closeParenTok.Type.String())
	}

	peek, err := p.peek()
	if err != nil {
		return nil, err
	}
	if peek.Type != EQUAL {
		return method, nil
	}

	// consume '='
	_, err = p.next()
	if err != nil {
		return nil, err
	}

	peek, err = p.peek()
	if err != nil {
		return nil, err
	}
	if peek.Type != CLOSE_ANGLE {
		return nil, NewError(peek, "expected '=>' in service method declaration, got %s", peek.Type.String())
	}

	// consume '>'
	_, err = p.next()
	if err != nil {
		return nil, err
	}

	openReturnParenTok, err := p.next()
	if err != nil {
		return nil, err
	}
	if openReturnParenTok.Type != OPEN_PAREN {
		return nil, NewError(openReturnParenTok, "expected '(' after '=>' in service method declaration, got %s", openReturnParenTok.Type.String())
	}

	for {
		peek, err := p.peek()
		if err != nil {
			return nil, err
		}
		if peek.Type == CLOSE_PAREN {
			break
		}

		ret, err := p.parseDeclNameTypePair()
		if err != nil {
			return nil, err
		}

		method.Returns = append(method.Returns, ret)

		peek, err = p.peek()
		if err != nil {
			return nil, err
		}
		if peek.Type == COMMA {
			_, _ = p.next() // consume ','
		}
	}

	closeReturnParenTok, err := p.next()
	if err != nil {
		return nil, err
	}
	if closeReturnParenTok.Type != CLOSE_PAREN {
		return nil, NewError(closeReturnParenTok, "expected ')' at the end of service method return types, got %s", closeReturnParenTok.Type.String())
	}

	return method, nil
}

func (p *Parser) parseServiceDecl() (*DeclService, error) {
	var err error

	serviceDecl := &DeclService{}

	// consume 'service'
	serviceDecl.Token, err = p.next()
	if err != nil {
		return nil, err
	}

	serviceDecl.Name, err = p.parseIdenExpr()
	if err != nil {
		return nil, err
	}

	openCurlyTok, err := p.next()
	if err != nil {
		return nil, err
	}
	if openCurlyTok.Type != OPEN_CURLY {
		return nil, NewError(openCurlyTok, "expected '{' after identifier in service declaration, got %s", openCurlyTok.Type.String())
	}

	for {
		peek, err := p.peek()
		if err != nil {
			return nil, err
		}
		if peek.Type == CLOSE_CURLY {
			break
		}

		method, err := p.parseDeclServiceMethod()
		if err != nil {
			return nil, err
		}

		serviceDecl.Methods = append(serviceDecl.Methods, method)
	}

	closeCurlyTok, err := p.next()
	if err != nil {
		return nil, err
	}
	if closeCurlyTok.Type != CLOSE_CURLY {
		return nil, NewError(closeCurlyTok, "expected '}' at the end of service declaration, got %s", closeCurlyTok.Type.String())
	}

	serviceDecl.CloseCurly = closeCurlyTok

	return serviceDecl, nil
}

func (p *Parser) parseErrorDecl() (*DeclError, error) {
	var err error

	errorDecl := &DeclError{}

	// consume 'error'
	errorDecl.Token, err = p.next()
	if err != nil {
		return nil, err
	}

	errorDecl.Name, err = p.parseIdenExpr()
	if err != nil {
		return nil, err
	}

	openCurlyTok, err := p.next()
	if err != nil {
		return nil, err
	}
	if openCurlyTok.Type != OPEN_CURLY {
		return nil, NewError(openCurlyTok, "expected '{' after identifier in error declaration, got %s", openCurlyTok.Type.String())
	}

	for {
		peek, err := p.peek()
		if err != nil {
			return nil, err
		}
		if peek.Type == CLOSE_CURLY {
			break
		}

		target := peek
		switch target.Lit {
		case "Msg":
			if errorDecl.Msg != nil {
				return nil, NewError(target, "duplicate 'Msg' field in error declaration")
			}

			// consume 'Msg'
			_, err = p.next()
			if err != nil {
				return nil, err
			}

			equal, err := p.next()
			if err != nil {
				return nil, err
			}
			if equal.Type != EQUAL {
				return nil, NewError(equal, "expected '=' after 'Msg' in error declaration, got %s", equal.Type.String())
			}

			valueExpr, err := p.parseValueExpr()
			if err != nil {
				return nil, err
			}

			msgExpr, ok := valueExpr.(*ValueExprString)
			if !ok {
				return nil, NewError(getTokenFromNode(valueExpr), "expected string value for 'Msg' field in error declaration, got %T", valueExpr)
			}

			errorDecl.Msg = msgExpr

		case "Code":
			if errorDecl.Code != nil {
				return nil, NewError(target, "duplicate 'Code' field in error declaration")
			}

			// consume 'Code'
			_, err = p.next()
			if err != nil {
				return nil, err
			}

			equal, err := p.next()
			if err != nil {
				return nil, err
			}
			if equal.Type != EQUAL {
				return nil, NewError(equal, "expected '=' after 'Code' in error declaration, got %s", equal.Type.String())
			}

			valueExpr, err := p.parseValueExpr()
			if err != nil {
				return nil, err
			}

			codeExpr, ok := valueExpr.(*ValueExprNumber)
			if !ok {
				return nil, NewError(getTokenFromNode(valueExpr), "expected number value for 'Code' field in error declaration, got %T", valueExpr)
			}

			errorDecl.Code = codeExpr

		default:
			return nil, NewError(target, "unexpected field '%s' in error declaration", target.Lit)
		}
	}

	closeCurlyTok, err := p.next()
	if err != nil {
		return nil, err
	}
	if closeCurlyTok.Type != CLOSE_CURLY {
		return nil, NewError(closeCurlyTok, "expected '}' at the end of error declaration, got %s", closeCurlyTok.Type.String())
	}

	errorDecl.CloseCurly = closeCurlyTok

	return errorDecl, nil
}
