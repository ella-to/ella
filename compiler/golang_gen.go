package compiler

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// GoGenerator transforms Ella AST to Go source code
type GoGenerator struct {
	program       *Program
	packageName   string
	enums         map[string]*DeclEnum  // map of enum name to enum declaration
	models        map[string]*DeclModel // map of model name to model declaration
	nextErrorCode int                   // next error code to assign
}

// NewGoGenerator creates a new Go code generator
func NewGoGenerator(program *Program, packageName string) *GoGenerator {
	g := &GoGenerator{
		program:       program,
		packageName:   packageName,
		enums:         make(map[string]*DeclEnum),
		models:        make(map[string]*DeclModel),
		nextErrorCode: 1000,
	}

	// Pre-process to collect enums and models for type resolution
	for _, node := range program.Nodes {
		switch n := node.(type) {
		case *DeclEnum:
			g.enums[n.Name.Name] = n
		case *DeclModel:
			g.models[n.Name.Name] = n
		}
	}

	return g
}

func (g *GoGenerator) GenerateToWriter(w io.Writer) error {
	file := &ast.File{
		Name:  ast.NewIdent(g.packageName),
		Decls: []ast.Decl{},
	}

	// Add imports
	imports := g.generateImports()
	if imports != nil {
		file.Decls = append(file.Decls, imports)
	}

	// Generate declarations
	for _, node := range g.program.Nodes {
		decls, err := g.generateNode(node)
		if err != nil {
			return err
		}
		file.Decls = append(file.Decls, decls...)
	}

	// Format and output
	fset := token.NewFileSet()
	if err := format.Node(w, fset, file); err != nil {
		return fmt.Errorf("failed to format Go code: %w", err)
	}

	return nil
}

// Generate produces Go source code from the Ella program
func (g *GoGenerator) Generate() (string, error) {
	var sb strings.Builder

	if err := g.GenerateToWriter(&sb); err != nil {
		return "", err
	}

	return sb.String(), nil
}

func (g *GoGenerator) generateImports() *ast.GenDecl {
	// Analyze what imports are needed
	hasServices := false
	hasErrors := false
	hasEnums := false
	needsTime := false

	for _, node := range g.program.Nodes {
		switch n := node.(type) {
		case *DeclService:
			hasServices = true
			// Check if any method args or returns use timestamp
			for _, m := range n.Methods {
				for _, arg := range m.Args {
					if g.typeNeedsTime(arg.Type) {
						needsTime = true
					}
				}
				for _, ret := range m.Returns {
					if g.typeNeedsTime(ret.Type) {
						needsTime = true
					}
				}
			}
		case *DeclModel:
			// Check if any field uses timestamp
			for _, f := range n.Fields {
				if g.typeNeedsTime(f.Type) {
					needsTime = true
				}
			}
		case *DeclError:
			hasErrors = true
		case *DeclEnum:
			hasEnums = true
		case *ConstDecl:
			// Check if const uses time units (ms, s, m, h)
			if num, ok := n.Assignment.Value.(*ValueExprNumber); ok && num.Type != nil {
				switch num.Type.Name {
				case "ms", "s", "m", "h":
					needsTime = true
				}
			}
		}
	}

	imports := []string{}

	if hasServices {
		imports = append(imports, "context")
		imports = append(imports, "encoding/json")
	} else if hasEnums {
		// Enums need encoding/json for MarshalJSON/UnmarshalJSON
		imports = append(imports, "encoding/json")
	}

	if hasErrors || hasEnums {
		imports = append(imports, "fmt")
	}

	if needsTime {
		imports = append(imports, "time")
	}

	if hasServices || hasErrors {
		imports = append(imports, "ella.to/jsonrpc")
	}

	if len(imports) == 0 {
		return nil
	}

	specs := make([]ast.Spec, 0, len(imports))
	for _, imp := range imports {
		specs = append(specs, &ast.ImportSpec{
			Path: &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(imp)},
		})
	}

	return &ast.GenDecl{
		Tok:    token.IMPORT,
		Lparen: token.Pos(1),
		Specs:  specs,
		Rparen: token.Pos(1),
	}
}

// typeNeedsTime checks if a type requires the time package
func (g *GoGenerator) typeNeedsTime(t DeclType) bool {
	switch dt := t.(type) {
	case *DeclTimestampType:
		return true
	case *DeclArrayType:
		return g.typeNeedsTime(dt.Type.(DeclType))
	case *DeclMapType:
		return g.typeNeedsTime(dt.KeyType.(DeclType)) || g.typeNeedsTime(dt.ValueType.(DeclType))
	default:
		return false
	}
}

func (g *GoGenerator) generateNode(node Node) ([]ast.Decl, error) {
	switch n := node.(type) {
	case *ConstDecl:
		return g.generateConst(n)
	case *DeclEnum:
		return g.generateEnum(n)
	case *DeclModel:
		return g.generateModel(n)
	case *DeclService:
		return g.generateService(n)
	case *DeclError:
		return g.generateError(n)
	default:
		return nil, fmt.Errorf("unknown node type: %T", node)
	}
}

// generateConst generates Go const declaration or function for template strings
func (g *GoGenerator) generateConst(c *ConstDecl) ([]ast.Decl, error) {
	// Check if this is a string value with template placeholders like {{name}}
	if strExpr, ok := c.Assignment.Value.(*ValueExprString); ok {
		strValue := strExpr.Token.Lit
		if hasTemplatePlaceholders(strValue) {
			return g.generateConstTemplateFunc(c.Assignment.Name.Name, strValue)
		}
	}

	value, err := g.exprToGoExpr(c.Assignment.Value)
	if err != nil {
		return nil, err
	}

	return []ast.Decl{
		&ast.GenDecl{
			Tok: token.CONST,
			Specs: []ast.Spec{
				&ast.ValueSpec{
					Names:  []*ast.Ident{ast.NewIdent(c.Assignment.Name.Name)},
					Values: []ast.Expr{value},
				},
			},
		},
	}, nil
}

// templatePlaceholderRegex matches {{name}} patterns
var templatePlaceholderRegex = regexp.MustCompile(`\{\{(\w+)\}\}`)

// hasTemplatePlaceholders checks if a string contains {{name}} patterns
func hasTemplatePlaceholders(s string) bool {
	return templatePlaceholderRegex.MatchString(s)
}

// generateConstTemplateFunc generates a function for template strings
// e.g., "user.{{userId}}.created" becomes func TopicUserCreated(userId string) string { return "user." + userId + ".created" }
func (g *GoGenerator) generateConstTemplateFunc(name string, template string) ([]ast.Decl, error) {
	// Find all placeholders
	matches := templatePlaceholderRegex.FindAllStringSubmatch(template, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no template placeholders found in %s", name)
	}

	// Extract unique parameter names in order
	seenParams := make(map[string]bool)
	var params []string
	for _, match := range matches {
		paramName := match[1]
		if !seenParams[paramName] {
			seenParams[paramName] = true
			params = append(params, paramName)
		}
	}

	// Build the function parameters
	funcParams := &ast.FieldList{
		List: make([]*ast.Field, len(params)),
	}
	for i, param := range params {
		funcParams.List[i] = &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(param)},
			Type:  ast.NewIdent("string"),
		}
	}

	// Build the return expression: "prefix" + param1 + "middle" + param2 + "suffix"
	returnExpr := buildTemplateReturnExpr(template, templatePlaceholderRegex)

	// Create the function declaration
	funcDecl := &ast.FuncDecl{
		Name: ast.NewIdent(name),
		Type: &ast.FuncType{
			Params: funcParams,
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: ast.NewIdent("string")},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{returnExpr},
				},
			},
		},
	}

	return []ast.Decl{funcDecl}, nil
}

// buildTemplateReturnExpr builds a concatenation expression from a template string
// e.g., "user.{{userId}}.created" becomes "user." + userId + ".created"
func buildTemplateReturnExpr(template string, re *regexp.Regexp) ast.Expr {
	// Split the template by placeholders
	parts := re.Split(template, -1)
	matches := re.FindAllStringSubmatch(template, -1)

	var exprs []ast.Expr

	for i, part := range parts {
		// Add the literal part if non-empty
		if part != "" {
			exprs = append(exprs, &ast.BasicLit{
				Kind:  token.STRING,
				Value: strconv.Quote(part),
			})
		}

		// Add the parameter reference if there's a corresponding match
		if i < len(matches) {
			paramName := matches[i][1]
			exprs = append(exprs, ast.NewIdent(paramName))
		}
	}

	// If only one expression, return it directly
	if len(exprs) == 1 {
		return exprs[0]
	}

	// Build a chain of binary + expressions
	result := exprs[0]
	for i := 1; i < len(exprs); i++ {
		result = &ast.BinaryExpr{
			X:  result,
			Op: token.ADD,
			Y:  exprs[i],
		}
	}

	return result
}

// generateEnum generates Go type and const declarations for enum
func (g *GoGenerator) generateEnum(e *DeclEnum) ([]ast.Decl, error) {
	decls := []ast.Decl{}

	// Determine if enum is string or int based
	isStringEnum := g.isStringEnum(e)

	var baseType *ast.Ident
	if isStringEnum {
		baseType = ast.NewIdent("string")
	} else {
		baseType = ast.NewIdent("int")
	}

	// Type declaration: type EnumName string/int
	typeDecl := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(e.Name.Name),
				Type: baseType,
			},
		},
	}
	decls = append(decls, typeDecl)

	// Const declarations for enum values
	if len(e.Values) > 0 {
		specs := make([]ast.Spec, 0, len(e.Values))

		// Track the next auto-increment value for int enums
		nextIntValue := int64(0)

		for _, v := range e.Values {
			var value ast.Expr

			if v.IsDefined {
				if isStringEnum {
					// String enum with explicit value
					var err error
					value, err = g.exprToGoExpr(v.Value)
					if err != nil {
						return nil, err
					}
				} else {
					// Int enum with explicit value - parse and track for auto-increment
					if numExpr, ok := v.Value.(*ValueExprNumber); ok {
						numVal, err := strconv.ParseInt(numExpr.Token.Lit, 10, 64)
						if err != nil {
							return nil, fmt.Errorf("invalid enum value: %s", numExpr.Token.Lit)
						}
						value = &ast.BasicLit{Kind: token.INT, Value: strconv.FormatInt(numVal, 10)}
						nextIntValue = numVal + 1
					} else {
						var err error
						value, err = g.exprToGoExpr(v.Value)
						if err != nil {
							return nil, err
						}
					}
				}
			} else {
				// No explicit value - auto-generate
				if isStringEnum {
					value = &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(v.Name.Name)}
				} else {
					// Use auto-incremented value
					value = &ast.BasicLit{Kind: token.INT, Value: strconv.FormatInt(nextIntValue, 10)}
					nextIntValue++
				}
			}

			// Handle underscore placeholder
			var constName string
			if v.Name.Name == "_" {
				constName = "_"
			} else {
				constName = e.Name.Name + "_" + v.Name.Name
			}

			spec := &ast.ValueSpec{
				Names:  []*ast.Ident{ast.NewIdent(constName)},
				Type:   ast.NewIdent(e.Name.Name),
				Values: []ast.Expr{value},
			}
			specs = append(specs, spec)
		}

		constDecl := &ast.GenDecl{
			Tok:    token.CONST,
			Lparen: token.Pos(1),
			Specs:  specs,
			Rparen: token.Pos(1),
		}
		decls = append(decls, constDecl)
	}

	// Generate String() method for non-string enums
	if !isStringEnum {
		stringMethod := g.generateEnumStringMethod(e)
		decls = append(decls, stringMethod)
	}

	// Generate MarshalJSON method
	marshalMethod := g.generateEnumMarshalJSON(e, isStringEnum)
	decls = append(decls, marshalMethod)

	// Generate UnmarshalJSON method
	unmarshalMethod := g.generateEnumUnmarshalJSON(e, isStringEnum)
	decls = append(decls, unmarshalMethod)

	return decls, nil
}

// generateEnumStringMethod generates the String() method for int-based enums
func (g *GoGenerator) generateEnumStringMethod(e *DeclEnum) ast.Decl {
	enumName := e.Name.Name
	receiverName := strings.ToLower(string(enumName[0]))

	// Build switch cases
	cases := []ast.Stmt{}
	for _, v := range e.Values {
		// Skip underscore placeholders
		if v.Name.Name == "_" {
			continue
		}

		constName := enumName + "_" + v.Name.Name
		cases = append(cases, &ast.CaseClause{
			List: []ast.Expr{ast.NewIdent(constName)},
			Body: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(v.Name.Name)},
					},
				},
			},
		})
	}

	// Add default case
	cases = append(cases, &ast.CaseClause{
		Body: []ast.Stmt{
			&ast.ReturnStmt{
				Results: []ast.Expr{
					&ast.CallExpr{
						Fun: &ast.SelectorExpr{X: ast.NewIdent("fmt"), Sel: ast.NewIdent("Sprintf")},
						Args: []ast.Expr{
							&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(enumName + "(%d)")},
							ast.NewIdent(receiverName),
						},
					},
				},
			},
		},
	})

	return &ast.FuncDecl{
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent(receiverName)},
					Type:  ast.NewIdent(enumName),
				},
			},
		},
		Name: ast.NewIdent("String"),
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{
				List: []*ast.Field{{Type: ast.NewIdent("string")}},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.SwitchStmt{
					Tag:  ast.NewIdent(receiverName),
					Body: &ast.BlockStmt{List: cases},
				},
			},
		},
	}
}

// generateEnumMarshalJSON generates the MarshalJSON method for enums
func (g *GoGenerator) generateEnumMarshalJSON(e *DeclEnum, isStringEnum bool) ast.Decl {
	enumName := e.Name.Name
	receiverName := strings.ToLower(string(enumName[0]))

	var body *ast.BlockStmt

	if isStringEnum {
		// For string enums, just marshal the string value
		body = &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{X: ast.NewIdent("json"), Sel: ast.NewIdent("Marshal")},
							Args: []ast.Expr{
								&ast.CallExpr{
									Fun:  ast.NewIdent("string"),
									Args: []ast.Expr{ast.NewIdent(receiverName)},
								},
							},
						},
					},
				},
			},
		}
	} else {
		// For int enums, marshal as string using String() method
		body = &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{X: ast.NewIdent("json"), Sel: ast.NewIdent("Marshal")},
							Args: []ast.Expr{
								&ast.CallExpr{
									Fun:  &ast.SelectorExpr{X: ast.NewIdent(receiverName), Sel: ast.NewIdent("String")},
									Args: []ast.Expr{},
								},
							},
						},
					},
				},
			},
		}
	}

	return &ast.FuncDecl{
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent(receiverName)},
					Type:  ast.NewIdent(enumName),
				},
			},
		},
		Name: ast.NewIdent("MarshalJSON"),
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: &ast.ArrayType{Elt: ast.NewIdent("byte")}},
					{Type: ast.NewIdent("error")},
				},
			},
		},
		Body: body,
	}
}

// generateEnumUnmarshalJSON generates the UnmarshalJSON method for enums
func (g *GoGenerator) generateEnumUnmarshalJSON(e *DeclEnum, isStringEnum bool) ast.Decl {
	enumName := e.Name.Name
	receiverName := strings.ToLower(string(enumName[0]))

	stmts := []ast.Stmt{}

	// var str string (use "str" to avoid conflict with receiver)
	stmts = append(stmts, &ast.DeclStmt{
		Decl: &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{
				&ast.ValueSpec{
					Names: []*ast.Ident{ast.NewIdent("str")},
					Type:  ast.NewIdent("string"),
				},
			},
		},
	})

	// if err := json.Unmarshal(data, &str); err != nil { return err }
	stmts = append(stmts, &ast.IfStmt{
		Init: &ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent("err")},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{X: ast.NewIdent("json"), Sel: ast.NewIdent("Unmarshal")},
					Args: []ast.Expr{
						ast.NewIdent("data"),
						&ast.UnaryExpr{Op: token.AND, X: ast.NewIdent("str")},
					},
				},
			},
		},
		Cond: &ast.BinaryExpr{X: ast.NewIdent("err"), Op: token.NEQ, Y: ast.NewIdent("nil")},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("err")}},
			},
		},
	})

	// Build switch cases
	cases := []ast.Stmt{}
	for _, v := range e.Values {
		// Skip underscore placeholders
		if v.Name.Name == "_" {
			continue
		}

		constName := enumName + "_" + v.Name.Name

		// For string enums with explicit values, match the value; otherwise match the name
		var matchValue string
		if isStringEnum && v.IsDefined {
			if strVal, ok := v.Value.(*ValueExprString); ok {
				matchValue = strVal.Token.Lit
			} else {
				matchValue = v.Name.Name
			}
		} else {
			matchValue = v.Name.Name
		}

		cases = append(cases, &ast.CaseClause{
			List: []ast.Expr{
				&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(matchValue)},
			},
			Body: []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{&ast.StarExpr{X: ast.NewIdent(receiverName)}},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{ast.NewIdent(constName)},
				},
			},
		})
	}

	// Add default case with error
	cases = append(cases, &ast.CaseClause{
		Body: []ast.Stmt{
			&ast.ReturnStmt{
				Results: []ast.Expr{
					&ast.CallExpr{
						Fun: &ast.SelectorExpr{X: ast.NewIdent("fmt"), Sel: ast.NewIdent("Errorf")},
						Args: []ast.Expr{
							&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote("unknown " + enumName + " value: %q")},
							ast.NewIdent("str"),
						},
					},
				},
			},
		},
	})

	// switch str { ... }
	stmts = append(stmts, &ast.SwitchStmt{
		Tag:  ast.NewIdent("str"),
		Body: &ast.BlockStmt{List: cases},
	})

	// return nil
	stmts = append(stmts, &ast.ReturnStmt{
		Results: []ast.Expr{ast.NewIdent("nil")},
	})

	return &ast.FuncDecl{
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent(receiverName)},
					Type:  &ast.StarExpr{X: ast.NewIdent(enumName)},
				},
			},
		},
		Name: ast.NewIdent("UnmarshalJSON"),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("data")},
						Type:  &ast.ArrayType{Elt: ast.NewIdent("byte")},
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{{Type: ast.NewIdent("error")}},
			},
		},
		Body: &ast.BlockStmt{List: stmts},
	}
}

// isStringEnum determines if an enum should be string-based
func (g *GoGenerator) isStringEnum(e *DeclEnum) bool {
	for _, v := range e.Values {
		if v.IsDefined {
			if _, ok := v.Value.(*ValueExprString); ok {
				return true
			}
		}
	}
	return false
}

// generateModel generates Go struct declaration
func (g *GoGenerator) generateModel(m *DeclModel) ([]ast.Decl, error) {
	fields := &ast.FieldList{List: []*ast.Field{}}

	// Handle extends (embedded structs)
	for _, ext := range m.Extends {
		fields.List = append(fields.List, &ast.Field{
			Type: ast.NewIdent(ext.Name),
		})
	}

	// Handle fields
	for _, f := range m.Fields {
		fieldType, err := g.declTypeToGoType(f.Type)
		if err != nil {
			return nil, err
		}

		jsonTag := g.toJSONTag(f.Name.Name, f.Options)

		field := &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(f.Name.Name)},
			Type:  fieldType,
			Tag:   &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("`json:%q`", jsonTag)},
		}
		fields.List = append(fields.List, field)
	}

	return []ast.Decl{
		&ast.GenDecl{
			Tok: token.TYPE,
			Specs: []ast.Spec{
				&ast.TypeSpec{
					Name: ast.NewIdent(m.Name.Name),
					Type: &ast.StructType{Fields: fields},
				},
			},
		},
	}, nil
}

// toJSONTag creates JSON tag with camelCase naming
func (g *GoGenerator) toJSONTag(name string, options []*AssignmentStmt) string {
	// Check if json option is explicitly set to false
	for _, opt := range options {
		if strings.ToLower(opt.Name.Name) == "json" {
			if vb, ok := opt.Value.(*ValueExprBool); ok && vb.Token.Lit == "false" {
				return "-"
			}
		}
	}
	return toCamelCase(name)
}

// toCamelCase converts a PascalCase string to camelCase
func toCamelCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// declTypeToGoType converts Ella type to Go AST type
func (g *GoGenerator) declTypeToGoType(t DeclType) (ast.Expr, error) {
	switch dt := t.(type) {
	case *DeclStringType:
		return ast.NewIdent("string"), nil
	case *DeclNumberType:
		return ast.NewIdent(dt.Name.Name), nil
	case *DeclBoolType:
		return ast.NewIdent("bool"), nil
	case *DeclByteType:
		return ast.NewIdent("byte"), nil
	case *DeclAnyType:
		return ast.NewIdent("any"), nil
	case *DeclTimestampType:
		return &ast.SelectorExpr{
			X:   ast.NewIdent("time"),
			Sel: ast.NewIdent("Time"),
		}, nil
	case *DeclArrayType:
		elemType, err := g.declTypeToGoType(dt.Type.(DeclType))
		if err != nil {
			return nil, err
		}
		return &ast.ArrayType{Elt: elemType}, nil
	case *DeclMapType:
		keyType, err := g.declTypeToGoType(dt.KeyType.(DeclType))
		if err != nil {
			return nil, err
		}
		valueType, err := g.declTypeToGoType(dt.ValueType.(DeclType))
		if err != nil {
			return nil, err
		}
		return &ast.MapType{Key: keyType, Value: valueType}, nil
	case *DeclCustomType:
		typeName := dt.Name.Name

		// Check if it's an enum
		if _, ok := g.enums[typeName]; ok {
			return ast.NewIdent(typeName), nil
		}

		// Check if it's a model (use pointer)
		if _, ok := g.models[typeName]; ok {
			return &ast.StarExpr{X: ast.NewIdent(typeName)}, nil
		}

		// Default to the type name as-is
		return ast.NewIdent(typeName), nil
	default:
		return nil, fmt.Errorf("unknown type: %T", t)
	}
}

// generateService generates Go interface, client, and server for a service
func (g *GoGenerator) generateService(s *DeclService) ([]ast.Decl, error) {
	decls := []ast.Decl{}

	// Generate interface
	interfaceDecl, err := g.generateServiceInterface(s)
	if err != nil {
		return nil, err
	}
	decls = append(decls, interfaceDecl)

	// Generate server implementation
	serverDecls, err := g.generateServiceServer(s)
	if err != nil {
		return nil, err
	}
	decls = append(decls, serverDecls...)

	// Generate client implementation
	clientDecls, err := g.generateServiceClient(s)
	if err != nil {
		return nil, err
	}
	decls = append(decls, clientDecls...)

	return decls, nil
}

// generateServiceInterface generates the service interface
func (g *GoGenerator) generateServiceInterface(s *DeclService) (ast.Decl, error) {
	methods := &ast.FieldList{List: []*ast.Field{}}

	for _, m := range s.Methods {
		methodType, err := g.methodToFuncType(m)
		if err != nil {
			return nil, err
		}
		methods.List = append(methods.List, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(m.Name.Name)},
			Type:  methodType,
		})
	}

	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(s.Name.Name),
				Type: &ast.InterfaceType{Methods: methods},
			},
		},
	}, nil
}

// methodToFuncType converts a service method to Go function type
func (g *GoGenerator) methodToFuncType(m *DeclServiceMethod) (*ast.FuncType, error) {
	// Parameters: always starts with context.Context
	params := &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{ast.NewIdent("ctx")},
				Type: &ast.SelectorExpr{
					X:   ast.NewIdent("context"),
					Sel: ast.NewIdent("Context"),
				},
			},
		},
	}

	for _, arg := range m.Args {
		argType, err := g.declTypeToGoType(arg.Type)
		if err != nil {
			return nil, err
		}
		params.List = append(params.List, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(arg.Name.Name)},
			Type:  argType,
		})
	}

	// Results: return values + error
	results := &ast.FieldList{List: []*ast.Field{}}
	for _, ret := range m.Returns {
		retType, err := g.declTypeToGoType(ret.Type)
		if err != nil {
			return nil, err
		}
		results.List = append(results.List, &ast.Field{
			Type: retType,
		})
	}
	// Always append error as last return type
	results.List = append(results.List, &ast.Field{
		Type: ast.NewIdent("error"),
	})

	return &ast.FuncType{Params: params, Results: results}, nil
}

// generateServiceServer generates server implementation
func (g *GoGenerator) generateServiceServer(s *DeclService) ([]ast.Decl, error) {
	decls := []ast.Decl{}
	serverTypeName := toLowerFirst(s.Name.Name) + "Server"

	// Server struct
	serverStruct := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(serverTypeName),
				Type: &ast.StructType{
					Fields: &ast.FieldList{
						List: []*ast.Field{
							{
								Names: []*ast.Ident{ast.NewIdent("impl")},
								Type:  ast.NewIdent(s.Name.Name),
							},
						},
					},
				},
			},
		},
	}
	decls = append(decls, serverStruct)

	// Server methods
	for _, m := range s.Methods {
		methodDecl := g.generateServerMethod(s, m, serverTypeName)
		decls = append(decls, methodDecl)
	}

	// Register function
	registerFunc := g.generateRegisterFunc(s, serverTypeName)
	decls = append(decls, registerFunc)

	return decls, nil
}

func (g *GoGenerator) generateServerMethod(s *DeclService, m *DeclServiceMethod, serverTypeName string) ast.Decl {
	stmts := []ast.Stmt{}

	// var Err error
	stmts = append(stmts, &ast.DeclStmt{
		Decl: &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{
				&ast.ValueSpec{
					Names: []*ast.Ident{ast.NewIdent("Err")},
					Type:  ast.NewIdent("error"),
				},
			},
		},
	})

	// Input struct
	if len(m.Args) > 0 {
		inputFields := &ast.FieldList{List: []*ast.Field{}}
		for _, arg := range m.Args {
			argType, _ := g.declTypeToGoType(arg.Type)
			inputFields.List = append(inputFields.List, &ast.Field{
				Names: []*ast.Ident{ast.NewIdent(toTitle(arg.Name.Name))},
				Type:  argType,
				Tag:   &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("`json:%q`", toCamelCase(arg.Name.Name))},
			})
		}

		stmts = append(stmts, &ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("Input")},
						Type:  &ast.StructType{Fields: inputFields},
					},
				},
			},
		})

		// Unmarshal input
		stmts = append(stmts, &ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent("Err")},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   ast.NewIdent("json"),
						Sel: ast.NewIdent("Unmarshal"),
					},
					Args: []ast.Expr{
						&ast.SelectorExpr{X: ast.NewIdent("req"), Sel: ast.NewIdent("Params")},
						&ast.UnaryExpr{Op: token.AND, X: ast.NewIdent("Input")},
					},
				},
			},
		})

		// Error check
		stmts = append(stmts, &ast.IfStmt{
			Cond: &ast.BinaryExpr{X: ast.NewIdent("Err"), Op: token.NEQ, Y: ast.NewIdent("nil")},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ReturnStmt{
						Results: []ast.Expr{
							&ast.CallExpr{
								Fun: &ast.SelectorExpr{X: ast.NewIdent("req"), Sel: ast.NewIdent("CreateErrorResponse")},
								Args: []ast.Expr{
									&ast.CallExpr{
										Fun: &ast.SelectorExpr{X: ast.NewIdent("jsonrpc"), Sel: ast.NewIdent("NewError")},
										Args: []ast.Expr{
											&ast.SelectorExpr{X: ast.NewIdent("jsonrpc"), Sel: ast.NewIdent("InvalidParams")},
											&ast.BasicLit{Kind: token.STRING, Value: `"invalid input: %v"`},
											ast.NewIdent("Err"),
										},
									},
								},
							},
						},
					},
				},
			},
		})
	}

	// Output struct if there are returns
	if len(m.Returns) > 0 {
		outputFields := &ast.FieldList{List: []*ast.Field{}}
		for _, ret := range m.Returns {
			retType, _ := g.declTypeToGoType(ret.Type)
			outputFields.List = append(outputFields.List, &ast.Field{
				Names: []*ast.Ident{ast.NewIdent(toTitle(ret.Name.Name))},
				Type:  retType,
				Tag:   &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("`json:%q`", toCamelCase(ret.Name.Name))},
			})
		}

		stmts = append(stmts, &ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("Output")},
						Type:  &ast.StructType{Fields: outputFields},
					},
				},
			},
		})
	}

	// Call impl method
	callArgs := []ast.Expr{ast.NewIdent("ctx")}
	for _, arg := range m.Args {
		callArgs = append(callArgs, &ast.SelectorExpr{
			X:   ast.NewIdent("Input"),
			Sel: ast.NewIdent(toTitle(arg.Name.Name)),
		})
	}

	lhs := []ast.Expr{}
	for _, ret := range m.Returns {
		lhs = append(lhs, &ast.SelectorExpr{
			X:   ast.NewIdent("Output"),
			Sel: ast.NewIdent(toTitle(ret.Name.Name)),
		})
	}
	lhs = append(lhs, ast.NewIdent("Err"))

	stmts = append(stmts, &ast.AssignStmt{
		Lhs: lhs,
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   &ast.SelectorExpr{X: ast.NewIdent("s"), Sel: ast.NewIdent("impl")},
					Sel: ast.NewIdent(m.Name.Name),
				},
				Args: callArgs,
			},
		},
	})

	// Error check after impl call
	stmts = append(stmts, &ast.IfStmt{
		Cond: &ast.BinaryExpr{X: ast.NewIdent("Err"), Op: token.NEQ, Y: ast.NewIdent("nil")},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.CallExpr{
							Fun:  &ast.SelectorExpr{X: ast.NewIdent("req"), Sel: ast.NewIdent("CreateErrorResponse")},
							Args: []ast.Expr{ast.NewIdent("Err")},
						},
					},
				},
			},
		},
	})

	// Return response
	var responseArg ast.Expr
	if len(m.Returns) > 0 {
		responseArg = ast.NewIdent("Output")
	} else {
		responseArg = ast.NewIdent("nil")
	}

	stmts = append(stmts, &ast.ReturnStmt{
		Results: []ast.Expr{
			&ast.CallExpr{
				Fun:  &ast.SelectorExpr{X: ast.NewIdent("req"), Sel: ast.NewIdent("CreateResponse")},
				Args: []ast.Expr{responseArg},
			},
		},
	})

	return &ast.FuncDecl{
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent("s")},
					Type:  &ast.StarExpr{X: ast.NewIdent(serverTypeName)},
				},
			},
		},
		Name: ast.NewIdent(m.Name.Name),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("ctx")},
						Type:  &ast.SelectorExpr{X: ast.NewIdent("context"), Sel: ast.NewIdent("Context")},
					},
					{
						Names: []*ast.Ident{ast.NewIdent("req")},
						Type:  &ast.StarExpr{X: &ast.SelectorExpr{X: ast.NewIdent("jsonrpc"), Sel: ast.NewIdent("Request")}},
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: &ast.StarExpr{X: &ast.SelectorExpr{X: ast.NewIdent("jsonrpc"), Sel: ast.NewIdent("Response")}}},
				},
			},
		},
		Body: &ast.BlockStmt{List: stmts},
	}
}

func (g *GoGenerator) generateRegisterFunc(s *DeclService, serverTypeName string) ast.Decl {
	stmts := []ast.Stmt{}

	// s := &serverTypeName{impl: srv}
	stmts = append(stmts, &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent("s")},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			&ast.UnaryExpr{
				Op: token.AND,
				X: &ast.CompositeLit{
					Type: ast.NewIdent(serverTypeName),
					Elts: []ast.Expr{
						&ast.KeyValueExpr{
							Key:   ast.NewIdent("impl"),
							Value: ast.NewIdent("srv"),
						},
					},
				},
			},
		},
	})

	// Register each method
	for _, m := range s.Methods {
		stmts = append(stmts, &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{X: ast.NewIdent("r"), Sel: ast.NewIdent("RegisterHandle")},
				Args: []ast.Expr{
					&ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf(`"%s.%s"`, s.Name.Name, m.Name.Name)},
					&ast.SelectorExpr{X: ast.NewIdent("s"), Sel: ast.NewIdent(m.Name.Name)},
				},
			},
		})
	}

	return &ast.FuncDecl{
		Name: ast.NewIdent("Register" + s.Name.Name + "Server"),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("r")},
						Type:  ast.NewIdent("HandleRegistry"),
					},
					{
						Names: []*ast.Ident{ast.NewIdent("srv")},
						Type:  ast.NewIdent(s.Name.Name),
					},
				},
			},
		},
		Body: &ast.BlockStmt{List: stmts},
	}
}

// generateServiceClient generates client implementation
func (g *GoGenerator) generateServiceClient(s *DeclService) ([]ast.Decl, error) {
	decls := []ast.Decl{}
	clientTypeName := toLowerFirst(s.Name.Name) + "Client"

	// Client struct
	clientStruct := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(clientTypeName),
				Type: &ast.StructType{
					Fields: &ast.FieldList{
						List: []*ast.Field{
							{
								Names: []*ast.Ident{ast.NewIdent("caller")},
								Type:  &ast.SelectorExpr{X: ast.NewIdent("jsonrpc"), Sel: ast.NewIdent("Caller")},
							},
						},
					},
				},
			},
		},
	}
	decls = append(decls, clientStruct)

	// Interface compliance check
	complianceCheck := &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{ast.NewIdent("_")},
				Type:  ast.NewIdent(s.Name.Name),
				Values: []ast.Expr{
					&ast.CallExpr{
						Fun:  &ast.ParenExpr{X: &ast.StarExpr{X: ast.NewIdent(clientTypeName)}},
						Args: []ast.Expr{ast.NewIdent("nil")},
					},
				},
			},
		},
	}
	decls = append(decls, complianceCheck)

	// Create function
	createFunc := &ast.FuncDecl{
		Name: ast.NewIdent("Create" + s.Name.Name + "Client"),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{ast.NewIdent("caller")},
						Type:  &ast.SelectorExpr{X: ast.NewIdent("jsonrpc"), Sel: ast.NewIdent("Caller")},
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: ast.NewIdent(s.Name.Name)},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.UnaryExpr{
							Op: token.AND,
							X: &ast.CompositeLit{
								Type: ast.NewIdent(clientTypeName),
								Elts: []ast.Expr{
									&ast.KeyValueExpr{
										Key:   ast.NewIdent("caller"),
										Value: ast.NewIdent("caller"),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	decls = append(decls, createFunc)

	// Client methods
	for _, m := range s.Methods {
		methodDecl, err := g.generateClientMethod(s, m, clientTypeName)
		if err != nil {
			return nil, err
		}
		decls = append(decls, methodDecl)
	}

	return decls, nil
}

func (g *GoGenerator) generateClientMethod(s *DeclService, m *DeclServiceMethod, clientTypeName string) (ast.Decl, error) {
	stmts := []ast.Stmt{}

	// Input struct
	if len(m.Args) > 0 {
		inputFields := &ast.FieldList{List: []*ast.Field{}}
		for _, arg := range m.Args {
			argType, _ := g.declTypeToGoType(arg.Type)
			inputFields.List = append(inputFields.List, &ast.Field{
				Names: []*ast.Ident{ast.NewIdent(toTitle(arg.Name.Name))},
				Type:  argType,
				Tag:   &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("`json:%q`", toCamelCase(arg.Name.Name))},
			})
		}

		stmts = append(stmts, &ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("In")},
						Type:  &ast.StructType{Fields: inputFields},
					},
				},
			},
		})

		// Assign input values
		for _, arg := range m.Args {
			stmts = append(stmts, &ast.AssignStmt{
				Lhs: []ast.Expr{&ast.SelectorExpr{X: ast.NewIdent("In"), Sel: ast.NewIdent(toTitle(arg.Name.Name))}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{ast.NewIdent(arg.Name.Name)},
			})
		}
	}

	// Call
	var callArg ast.Expr
	if len(m.Args) > 0 {
		callArg = &ast.UnaryExpr{Op: token.AND, X: ast.NewIdent("In")}
	} else {
		callArg = ast.NewIdent("nil")
	}

	stmts = append(stmts, &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent("Results"), ast.NewIdent("err")},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   &ast.SelectorExpr{X: ast.NewIdent("c"), Sel: ast.NewIdent("caller")},
					Sel: ast.NewIdent("Call"),
				},
				Args: []ast.Expr{
					ast.NewIdent("ctx"),
					&ast.CallExpr{
						Fun: &ast.SelectorExpr{X: ast.NewIdent("jsonrpc"), Sel: ast.NewIdent("WithRequest")},
						Args: []ast.Expr{
							&ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf(`"%s.%s"`, s.Name.Name, m.Name.Name)},
							callArg,
							ast.NewIdent("false"),
						},
					},
				},
			},
		},
	})

	// Build zero values for returns
	zeroReturns := g.buildZeroReturns(m.Returns)
	zeroReturns = append(zeroReturns, ast.NewIdent("err"))

	// Error check
	stmts = append(stmts, &ast.IfStmt{
		Cond: &ast.BinaryExpr{X: ast.NewIdent("err"), Op: token.NEQ, Y: ast.NewIdent("nil")},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{Results: zeroReturns},
			},
		},
	})

	// Check results length
	stmts = append(stmts, &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  &ast.CallExpr{Fun: ast.NewIdent("len"), Args: []ast.Expr{ast.NewIdent("Results")}},
			Op: token.NEQ,
			Y:  &ast.BasicLit{Kind: token.INT, Value: "1"},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: append(g.buildZeroReturns(m.Returns), &ast.CallExpr{
						Fun: &ast.SelectorExpr{X: ast.NewIdent("jsonrpc"), Sel: ast.NewIdent("NewError")},
						Args: []ast.Expr{
							&ast.SelectorExpr{X: ast.NewIdent("jsonrpc"), Sel: ast.NewIdent("InternalError")},
							&ast.BasicLit{Kind: token.STRING, Value: `"expected 1 result, got %d"`},
							&ast.CallExpr{Fun: ast.NewIdent("len"), Args: []ast.Expr{ast.NewIdent("Results")}},
						},
					}),
				},
			},
		},
	})

	// Get Result
	stmts = append(stmts, &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent("Result")},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{&ast.IndexExpr{X: ast.NewIdent("Results"), Index: &ast.BasicLit{Kind: token.INT, Value: "0"}}},
	})

	// Check error in result
	stmts = append(stmts, &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  &ast.SelectorExpr{X: ast.NewIdent("Result"), Sel: ast.NewIdent("Error")},
			Op: token.NEQ,
			Y:  ast.NewIdent("nil"),
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: append(g.buildZeroReturns(m.Returns), &ast.SelectorExpr{X: ast.NewIdent("Result"), Sel: ast.NewIdent("Error")}),
				},
			},
		},
	})

	// Output struct
	if len(m.Returns) > 0 {
		outputFields := &ast.FieldList{List: []*ast.Field{}}
		for _, ret := range m.Returns {
			retType, _ := g.declTypeToGoType(ret.Type)
			outputFields.List = append(outputFields.List, &ast.Field{
				Names: []*ast.Ident{ast.NewIdent(toTitle(ret.Name.Name))},
				Type:  retType,
				Tag:   &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("`json:%q`", toCamelCase(ret.Name.Name))},
			})
		}

		stmts = append(stmts, &ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent("Output")},
						Type:  &ast.StructType{Fields: outputFields},
					},
				},
			},
		})

		// Unmarshal
		stmts = append(stmts, &ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent("err")},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{X: ast.NewIdent("json"), Sel: ast.NewIdent("Unmarshal")},
					Args: []ast.Expr{
						&ast.SelectorExpr{X: ast.NewIdent("Result"), Sel: ast.NewIdent("Result")},
						&ast.UnaryExpr{Op: token.AND, X: ast.NewIdent("Output")},
					},
				},
			},
		})

		// Error check
		stmts = append(stmts, &ast.IfStmt{
			Cond: &ast.BinaryExpr{X: ast.NewIdent("err"), Op: token.NEQ, Y: ast.NewIdent("nil")},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ReturnStmt{
						Results: append(g.buildZeroReturns(m.Returns), &ast.CallExpr{
							Fun: &ast.SelectorExpr{X: ast.NewIdent("jsonrpc"), Sel: ast.NewIdent("NewError")},
							Args: []ast.Expr{
								&ast.SelectorExpr{X: ast.NewIdent("jsonrpc"), Sel: ast.NewIdent("InternalError")},
								&ast.BasicLit{Kind: token.STRING, Value: `"failed to decode response: %v"`},
								ast.NewIdent("err"),
							},
						}),
					},
				},
			},
		})
	}

	// Final return
	finalReturns := []ast.Expr{}
	for _, ret := range m.Returns {
		finalReturns = append(finalReturns, &ast.SelectorExpr{
			X:   ast.NewIdent("Output"),
			Sel: ast.NewIdent(toTitle(ret.Name.Name)),
		})
	}
	finalReturns = append(finalReturns, ast.NewIdent("nil"))

	stmts = append(stmts, &ast.ReturnStmt{Results: finalReturns})

	// Build function signature
	params := &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{ast.NewIdent("ctx")},
				Type:  &ast.SelectorExpr{X: ast.NewIdent("context"), Sel: ast.NewIdent("Context")},
			},
		},
	}
	for _, arg := range m.Args {
		argType, _ := g.declTypeToGoType(arg.Type)
		params.List = append(params.List, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(arg.Name.Name)},
			Type:  argType,
		})
	}

	results := &ast.FieldList{List: []*ast.Field{}}
	for _, ret := range m.Returns {
		retType, _ := g.declTypeToGoType(ret.Type)
		results.List = append(results.List, &ast.Field{Type: retType})
	}
	results.List = append(results.List, &ast.Field{Type: ast.NewIdent("error")})

	return &ast.FuncDecl{
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent("c")},
					Type:  &ast.StarExpr{X: ast.NewIdent(clientTypeName)},
				},
			},
		},
		Name: ast.NewIdent(m.Name.Name),
		Type: &ast.FuncType{Params: params, Results: results},
		Body: &ast.BlockStmt{List: stmts},
	}, nil
}

func (g *GoGenerator) buildZeroReturns(returns []*DeclNameTypePair) []ast.Expr {
	zeros := []ast.Expr{}
	for _, ret := range returns {
		zeros = append(zeros, g.zeroValue(ret.Type))
	}
	return zeros
}

func (g *GoGenerator) zeroValue(t DeclType) ast.Expr {
	switch dt := t.(type) {
	case *DeclStringType:
		return &ast.BasicLit{Kind: token.STRING, Value: `""`}
	case *DeclNumberType:
		return &ast.BasicLit{Kind: token.INT, Value: "0"}
	case *DeclBoolType:
		return ast.NewIdent("false")
	case *DeclByteType:
		return &ast.BasicLit{Kind: token.INT, Value: "0"}
	case *DeclAnyType:
		return ast.NewIdent("nil")
	case *DeclTimestampType:
		return &ast.CompositeLit{
			Type: &ast.SelectorExpr{X: ast.NewIdent("time"), Sel: ast.NewIdent("Time")},
		}
	case *DeclArrayType, *DeclMapType:
		return ast.NewIdent("nil")
	case *DeclCustomType:
		typeName := dt.Name.Name
		if enumDecl, ok := g.enums[typeName]; ok {
			if g.isStringEnum(enumDecl) {
				return &ast.BasicLit{Kind: token.STRING, Value: `""`}
			}
			return &ast.BasicLit{Kind: token.INT, Value: "0"}
		}
		return ast.NewIdent("nil")
	default:
		return ast.NewIdent("nil")
	}
}

// generateError generates custom error type
func (g *GoGenerator) generateError(e *DeclError) ([]ast.Decl, error) {
	decls := []ast.Decl{}

	// Determine error code
	var errorCode int
	if e.Code != nil {
		code, err := strconv.Atoi(e.Code.Token.Lit)
		if err != nil {
			return nil, fmt.Errorf("invalid error code: %s", e.Code.Token.Lit)
		}
		errorCode = code
	} else {
		errorCode = g.nextErrorCode
		g.nextErrorCode++
	}

	// Error variable using jsonrpc.NewError(code, message)
	errorVar := &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{ast.NewIdent(e.Name.Name)},
				Values: []ast.Expr{
					&ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ast.NewIdent("jsonrpc"),
							Sel: ast.NewIdent("NewError"),
						},
						Args: []ast.Expr{
							&ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(errorCode)},
							&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(e.Msg.Token.Lit)},
						},
					},
				},
			},
		},
	}
	decls = append(decls, errorVar)

	return decls, nil
}

// exprToGoExpr converts Ella expression to Go AST expression
func (g *GoGenerator) exprToGoExpr(expr Expr) (ast.Expr, error) {
	switch e := expr.(type) {
	case *ValueExprNumber:
		value := e.Token.Lit
		if e.Type != nil {
			// Handle size units like kb, mb, gb, etc.
			multiplier := int64(1)
			switch e.Type.Name {
			case "kb":
				multiplier = 1024
			case "mb":
				multiplier = 1024 * 1024
			case "gb":
				multiplier = 1024 * 1024 * 1024
			case "tb":
				multiplier = 1024 * 1024 * 1024 * 1024
			case "pb":
				multiplier = 1024 * 1024 * 1024 * 1024 * 1024
			case "eb":
				multiplier = 1024 * 1024 * 1024 * 1024 * 1024 * 1024
			case "ms":
				// milliseconds - return time.Duration
				return &ast.BinaryExpr{
					X:  &ast.BasicLit{Kind: token.INT, Value: value},
					Op: token.MUL,
					Y:  &ast.SelectorExpr{X: ast.NewIdent("time"), Sel: ast.NewIdent("Millisecond")},
				}, nil
			case "s":
				return &ast.BinaryExpr{
					X:  &ast.BasicLit{Kind: token.INT, Value: value},
					Op: token.MUL,
					Y:  &ast.SelectorExpr{X: ast.NewIdent("time"), Sel: ast.NewIdent("Second")},
				}, nil
			case "m":
				return &ast.BinaryExpr{
					X:  &ast.BasicLit{Kind: token.INT, Value: value},
					Op: token.MUL,
					Y:  &ast.SelectorExpr{X: ast.NewIdent("time"), Sel: ast.NewIdent("Minute")},
				}, nil
			case "h":
				return &ast.BinaryExpr{
					X:  &ast.BasicLit{Kind: token.INT, Value: value},
					Op: token.MUL,
					Y:  &ast.SelectorExpr{X: ast.NewIdent("time"), Sel: ast.NewIdent("Hour")},
				}, nil
			}
			if multiplier > 1 {
				numVal, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return nil, err
				}
				return &ast.BasicLit{Kind: token.INT, Value: strconv.FormatInt(numVal*multiplier, 10)}, nil
			}
		}
		return &ast.BasicLit{Kind: token.INT, Value: value}, nil
	case *ValueExprString:
		return &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(e.Token.Lit)}, nil
	case *ValueExprBool:
		return ast.NewIdent(e.Token.Lit), nil
	case *ValueExprNull:
		return ast.NewIdent("nil"), nil
	case *IdenExpr:
		return ast.NewIdent(e.Name), nil
	default:
		return nil, fmt.Errorf("unknown expression type: %T", expr)
	}
}

func toLowerFirst(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func toTitle(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// GenerateHelperTypes generates common helper types like HandleRegistry and Error
func (g *GoGenerator) GenerateHelperTypes() string {
	return `
// HandleRegistry is the interface for registering service handlers
type HandleRegistry interface {
	RegisterHandle(name string, handler jsonrpc.HandlerFunc)
}
`
}

// GenerateWithHelpers produces complete Go source with helper types
func (g *GoGenerator) GenerateWithHelpers() (string, error) {
	mainCode, err := g.Generate()
	if err != nil {
		return "", err
	}

	// Check if we have services or errors to add helper types
	hasServices := false
	hasErrors := false
	for _, node := range g.program.Nodes {
		if _, ok := node.(*DeclService); ok {
			hasServices = true
		}
		if _, ok := node.(*DeclError); ok {
			hasErrors = true
		}
	}

	if !hasServices && !hasErrors {
		return mainCode, nil
	}

	// Insert helper types before the closing of the file
	helpers := g.GenerateHelperTypes()

	// Find a good insertion point (after imports, before first real declaration)
	return mainCode + helpers, nil
}
