package compiler

import (
	"fmt"
	"strconv"
)

// Validator validates an Ella AST
type Validator struct {
	program  *Program
	consts   map[string]*ConstDecl
	enums    map[string]*DeclEnum
	models   map[string]*DeclModel
	services map[string]*DeclService
	errors   []error
}

// NewValidator creates a new validator
func NewValidator(program *Program) *Validator {
	return &Validator{
		program:  program,
		consts:   make(map[string]*ConstDecl),
		enums:    make(map[string]*DeclEnum),
		models:   make(map[string]*DeclModel),
		services: make(map[string]*DeclService),
		errors:   []error{},
	}
}

// Validate validates the program and returns any errors
func (v *Validator) Validate() []error {
	// First pass: collect all declarations
	v.collectDeclarations()

	// Second pass: validate each node
	for _, node := range v.program.Nodes {
		v.validateNode(node)
	}

	return v.errors
}

func (v *Validator) addError(token *Token, format string, args ...interface{}) {
	v.errors = append(v.errors, &Error{
		Token:  token,
		Reason: fmt.Sprintf(format, args...),
	})
}

func (v *Validator) collectDeclarations() {
	for _, node := range v.program.Nodes {
		switch n := node.(type) {
		case *ConstDecl:
			name := n.Assignment.Name.Name
			if existing, ok := v.consts[name]; ok {
				v.addError(n.Assignment.Name.Token, "duplicate const declaration '%s', previously declared at line %d", name, existing.Assignment.Name.Token.Pos.Line)
			} else {
				v.consts[name] = n
			}
			// Also check against other types
			v.checkNameConflict(name, n.Assignment.Name.Token, "const")

		case *DeclEnum:
			name := n.Name.Name
			if existing, ok := v.enums[name]; ok {
				v.addError(n.Name.Token, "duplicate enum declaration '%s', previously declared at line %d", name, existing.Name.Token.Pos.Line)
			} else {
				v.enums[name] = n
			}
			v.checkNameConflict(name, n.Name.Token, "enum")

		case *DeclModel:
			name := n.Name.Name
			if existing, ok := v.models[name]; ok {
				v.addError(n.Name.Token, "duplicate model declaration '%s', previously declared at line %d", name, existing.Name.Token.Pos.Line)
			} else {
				v.models[name] = n
			}
			v.checkNameConflict(name, n.Name.Token, "model")

		case *DeclService:
			name := n.Name.Name
			if existing, ok := v.services[name]; ok {
				v.addError(n.Name.Token, "duplicate service declaration '%s', previously declared at line %d", name, existing.Name.Token.Pos.Line)
			} else {
				v.services[name] = n
			}
			v.checkNameConflict(name, n.Name.Token, "service")
		}
	}
}

func (v *Validator) checkNameConflict(name string, token *Token, declType string) {
	// Check if name conflicts with other declaration types
	if declType != "const" {
		if existing, ok := v.consts[name]; ok {
			v.addError(token, "%s '%s' conflicts with const declared at line %d", declType, name, existing.Assignment.Name.Token.Pos.Line)
		}
	}
	if declType != "enum" {
		if existing, ok := v.enums[name]; ok {
			v.addError(token, "%s '%s' conflicts with enum declared at line %d", declType, name, existing.Name.Token.Pos.Line)
		}
	}
	if declType != "model" {
		if existing, ok := v.models[name]; ok {
			v.addError(token, "%s '%s' conflicts with model declared at line %d", declType, name, existing.Name.Token.Pos.Line)
		}
	}
	if declType != "service" {
		if existing, ok := v.services[name]; ok {
			v.addError(token, "%s '%s' conflicts with service declared at line %d", declType, name, existing.Name.Token.Pos.Line)
		}
	}
}

func (v *Validator) validateNode(node Node) {
	switch n := node.(type) {
	case *ConstDecl:
		v.validateConst(n)
	case *DeclEnum:
		v.validateEnum(n)
	case *DeclModel:
		v.validateModel(n)
	case *DeclService:
		v.validateService(n)
	case *DeclError:
		v.validateError(n)
	}
}

func (v *Validator) validateConst(c *ConstDecl) {
	// If the value is an identifier, it must reference another const
	if iden, ok := c.Assignment.Value.(*IdenExpr); ok {
		if _, exists := v.consts[iden.Name]; !exists {
			v.addError(iden.Token, "undefined const '%s' referenced in const '%s'", iden.Name, c.Assignment.Name.Name)
		}
	}
}

func (v *Validator) validateEnum(e *DeclEnum) {
	// Check for duplicate enum value names
	seenNames := make(map[string]*DeclEnumSet)
	for _, val := range e.Values {
		if existing, ok := seenNames[val.Name.Name]; ok {
			v.addError(val.Name.Token, "duplicate enum value name '%s' in enum '%s', previously declared at line %d", val.Name.Name, e.Name.Name, existing.Name.Token.Pos.Line)
		} else {
			seenNames[val.Name.Name] = val
		}
	}

	// Check for duplicate enum values (the actual assigned values)
	isStringEnum := v.isStringEnum(e)
	if isStringEnum {
		v.validateStringEnumValues(e)
	} else {
		v.validateIntEnumValues(e)
	}
}

// isStringEnum determines if an enum should be string-based
func (v *Validator) isStringEnum(e *DeclEnum) bool {
	for _, val := range e.Values {
		if val.IsDefined {
			if _, ok := val.Value.(*ValueExprString); ok {
				return true
			}
		}
	}
	return false
}

// validateStringEnumValues checks for duplicate string values in a string enum
func (v *Validator) validateStringEnumValues(e *DeclEnum) {
	seenValues := make(map[string]*DeclEnumSet)

	for _, val := range e.Values {
		// Skip underscore placeholders
		if val.Name.Name == "_" {
			continue
		}

		var strValue string
		if val.IsDefined {
			if strExpr, ok := val.Value.(*ValueExprString); ok {
				strValue = strExpr.Token.Lit
			} else {
				// Non-string value in a string enum
				v.addError(val.Name.Token, "enum '%s' value '%s' must be a string", e.Name.Name, val.Name.Name)
				continue
			}
		} else {
			// Default to the name
			strValue = val.Name.Name
		}

		if existing, ok := seenValues[strValue]; ok {
			v.addError(val.Name.Token, "duplicate enum value '%s' in enum '%s', same value as '%s' at line %d", strValue, e.Name.Name, existing.Name.Name, existing.Name.Token.Pos.Line)
		} else {
			seenValues[strValue] = val
		}
	}
}

// validateIntEnumValues checks for duplicate int values in an int enum
func (v *Validator) validateIntEnumValues(e *DeclEnum) {
	seenValues := make(map[int64]*DeclEnumSet)
	nextValue := int64(0)

	for _, val := range e.Values {
		// Skip underscore placeholders but still increment
		if val.Name.Name == "_" {
			if val.IsDefined {
				if numExpr, ok := val.Value.(*ValueExprNumber); ok {
					if num, err := strconv.ParseInt(numExpr.Token.Lit, 10, 64); err == nil {
						nextValue = num + 1
					}
				}
			} else {
				nextValue++
			}
			continue
		}

		var intValue int64
		if val.IsDefined {
			if numExpr, ok := val.Value.(*ValueExprNumber); ok {
				num, err := strconv.ParseInt(numExpr.Token.Lit, 10, 64)
				if err != nil {
					v.addError(val.Name.Token, "invalid number value '%s' in enum '%s'", numExpr.Token.Lit, e.Name.Name)
					continue
				}
				intValue = num
				nextValue = num + 1
			} else {
				// Non-number value in an int enum
				v.addError(val.Name.Token, "enum '%s' value '%s' must be a number", e.Name.Name, val.Name.Name)
				continue
			}
		} else {
			// Auto-increment
			intValue = nextValue
			nextValue++
		}

		if existing, ok := seenValues[intValue]; ok {
			v.addError(val.Name.Token, "duplicate enum value %d in enum '%s', same value as '%s' at line %d", intValue, e.Name.Name, existing.Name.Name, existing.Name.Token.Pos.Line)
		} else {
			seenValues[intValue] = val
		}
	}
}

func (v *Validator) validateModel(m *DeclModel) {
	// Check for duplicate field names
	seen := make(map[string]*DeclModelField)
	for _, field := range m.Fields {
		if existing, ok := seen[field.Name.Name]; ok {
			v.addError(field.Name.Token, "duplicate field '%s' in model '%s', previously declared at line %d", field.Name.Name, m.Name.Name, existing.Name.Token.Pos.Line)
		} else {
			seen[field.Name.Name] = field
		}

		// Validate field type
		v.validateType(field.Type, fmt.Sprintf("field '%s' in model '%s'", field.Name.Name, m.Name.Name))

		// Validate field options
		v.validateOptions(field.Options, fmt.Sprintf("field '%s' in model '%s'", field.Name.Name, m.Name.Name))
	}

	// Validate extends
	for _, ext := range m.Extends {
		if _, ok := v.models[ext.Name]; !ok {
			v.addError(ext.Token, "model '%s' extends unknown model '%s'", m.Name.Name, ext.Name)
		}
	}
}

func (v *Validator) validateService(s *DeclService) {
	// Check for duplicate method names
	seen := make(map[string]*DeclServiceMethod)
	for _, method := range s.Methods {
		if existing, ok := seen[method.Name.Name]; ok {
			v.addError(method.Name.Token, "duplicate method '%s' in service '%s', previously declared at line %d", method.Name.Name, s.Name.Name, existing.Name.Token.Pos.Line)
		} else {
			seen[method.Name.Name] = method
		}

		// Validate method args
		argNames := make(map[string]*DeclNameTypePair)
		for _, arg := range method.Args {
			if existing, ok := argNames[arg.Name.Name]; ok {
				v.addError(arg.Name.Token, "duplicate argument '%s' in method '%s.%s', previously declared at line %d", arg.Name.Name, s.Name.Name, method.Name.Name, existing.Name.Token.Pos.Line)
			} else {
				argNames[arg.Name.Name] = arg
			}
			v.validateType(arg.Type, fmt.Sprintf("argument '%s' in method '%s.%s'", arg.Name.Name, s.Name.Name, method.Name.Name))
		}

		// Validate method returns
		returnNames := make(map[string]*DeclNameTypePair)
		for _, ret := range method.Returns {
			if existing, ok := returnNames[ret.Name.Name]; ok {
				v.addError(ret.Name.Token, "duplicate return '%s' in method '%s.%s', previously declared at line %d", ret.Name.Name, s.Name.Name, method.Name.Name, existing.Name.Token.Pos.Line)
			} else {
				returnNames[ret.Name.Name] = ret
			}
			v.validateType(ret.Type, fmt.Sprintf("return '%s' in method '%s.%s'", ret.Name.Name, s.Name.Name, method.Name.Name))
		}

		// Validate method options
		v.validateOptions(method.Options, fmt.Sprintf("method '%s.%s'", s.Name.Name, method.Name.Name))
	}
}

func (v *Validator) validateError(e *DeclError) {
	// Error declarations are relatively simple, nothing special to validate
	// The code and message are already parsed correctly
}

func (v *Validator) validateType(t DeclType, context string) {
	switch dt := t.(type) {
	case *DeclStringType, *DeclNumberType, *DeclBoolType, *DeclByteType, *DeclTimestampType:
		// Built-in types are always valid
		return

	case *DeclCustomType:
		typeName := dt.Name.Name
		// Must be an enum or model
		if _, ok := v.enums[typeName]; ok {
			return
		}
		if _, ok := v.models[typeName]; ok {
			return
		}
		v.addError(dt.Name.Token, "unknown type '%s' in %s", typeName, context)

	case *DeclArrayType:
		v.validateType(dt.Type.(DeclType), context)

	case *DeclMapType:
		// Key type must be comparable (string or number)
		v.validateMapKeyType(dt.KeyType.(DeclType), context)
		v.validateType(dt.ValueType.(DeclType), context)
	}
}

func (v *Validator) validateMapKeyType(t DeclType, context string) {
	switch t.(type) {
	case *DeclStringType, *DeclNumberType:
		// Valid map key types
		return
	default:
		token := getTokenFromDeclType(t)
		v.addError(token, "map key type must be string or number in %s", context)
	}
}

func getTokenFromDeclType(t DeclType) *Token {
	switch dt := t.(type) {
	case *DeclStringType:
		return dt.Name.Token
	case *DeclNumberType:
		return dt.Name.Token
	case *DeclBoolType:
		return dt.Name.Token
	case *DeclByteType:
		return dt.Name.Token
	case *DeclTimestampType:
		return dt.Name.Token
	case *DeclCustomType:
		return dt.Name.Token
	case *DeclArrayType:
		return dt.Token
	case *DeclMapType:
		return dt.Token
	default:
		return nil
	}
}

func (v *Validator) validateOptions(options []*AssignmentStmt, context string) {
	for _, opt := range options {
		v.validateOptionValue(opt.Value, opt.Name.Token, context)
	}
}

func (v *Validator) validateOptionValue(value Expr, token *Token, context string) {
	switch val := value.(type) {
	case *ValueExprNumber, *ValueExprString, *ValueExprBool:
		// Valid option values
		return
	case *IdenExpr:
		// Must reference a const
		if _, ok := v.consts[val.Name]; !ok {
			v.addError(val.Token, "option value '%s' in %s must be a const, but '%s' is not defined", val.Name, context, val.Name)
		}
	default:
		v.addError(token, "option value in %s must be a number, string, bool, or const reference", context)
	}
}

// ValidateProgram is a convenience function to validate a program
func ValidateProgram(program *Program) []error {
	validator := NewValidator(program)
	return validator.Validate()
}
