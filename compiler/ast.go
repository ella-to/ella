package compiler

import (
	"fmt"
	"strings"
)

type Node interface {
	node()
	fmt.Stringer
}

func (*IdenExpr) node()          {}
func (*ConstDecl) node()         {}
func (*ValueExprBool) node()     {}
func (*ValueExprNull) node()     {}
func (*ValueExprNumber) node()   {}
func (*DeclAnyType) node()       {}
func (*ValueExprString) node()   {}
func (*DeclEnum) node()          {}
func (*DeclEnumSet) node()       {}
func (*DeclModel) node()         {}
func (*DeclModelField) node()    {}
func (*DeclCustomType) node()    {}
func (*DeclStringType) node()    {}
func (*DeclByteType) node()      {}
func (*DeclTimestampType) node() {}
func (*DeclNumberType) node()    {}
func (*DeclBoolType) node()      {}
func (*DeclArrayType) node()     {}
func (*DeclMapType) node()       {}
func (*AssignmentStmt) node()    {}
func (*DeclNameTypePair) node()  {}
func (*DeclServiceMethod) node() {}
func (*DeclService) node()       {}
func (*DeclError) node()         {}

type Decl interface {
	Node
	decl()
}

func (*ConstDecl) decl()         {}
func (*DeclEnum) decl()          {}
func (*DeclEnumSet) decl()       {}
func (*DeclModel) decl()         {}
func (*DeclModelField) decl()    {}
func (*DeclCustomType) decl()    {}
func (*DeclStringType) decl()    {}
func (*DeclByteType) decl()      {}
func (*DeclTimestampType) decl() {}
func (*DeclNumberType) decl()    {}
func (*DeclAnyType) decl()       {}
func (*DeclBoolType) decl()      {}
func (*DeclArrayType) decl()     {}
func (*DeclMapType) decl()       {}
func (*DeclNameTypePair) decl()  {}
func (*DeclServiceMethod) decl() {}
func (*DeclService) decl()       {}
func (*DeclError) decl()         {}

type DeclType interface {
	Decl
	declType()
}

func (*DeclCustomType) declType()    {}
func (*DeclStringType) declType()    {}
func (*DeclByteType) declType()      {}
func (*DeclTimestampType) declType() {}
func (*DeclNumberType) declType()    {}
func (*DeclAnyType) declType()       {}
func (*DeclBoolType) declType()      {}
func (*DeclArrayType) declType()     {}
func (*DeclMapType) declType()       {}

type Stmt interface {
	Node
	stmt()
}

func (*AssignmentStmt) stmt() {}

type Expr interface {
	Node
	expr()
}

func (*IdenExpr) expr()        {}
func (*ValueExprBool) expr()   {}
func (*ValueExprNull) expr()   {}
func (*ValueExprNumber) expr() {}
func (*ValueExprString) expr() {}

//
// AST Nodes
//

type IdenExpr struct {
	Token *Token
	Name  string
}

func (ie *IdenExpr) String() string {
	return ie.Name
}

type AssignmentStmt struct {
	Name  *IdenExpr
	Value Expr
}

func (ae *AssignmentStmt) String() string {
	return ae.Name.String() + " = " + ae.Value.String()
}

type ConstDecl struct {
	Token      *Token // 'const' token
	Assignment *AssignmentStmt
}

func (cd *ConstDecl) String() string {
	return cd.Token.Lit + " " + cd.Assignment.String()
}

type ValueExprNumber struct {
	Token *Token
	Type  *IdenExpr
}

func (ven *ValueExprNumber) String() string {
	var sb strings.Builder

	sb.WriteString(ven.Token.Lit)
	if ven.Type != nil {
		sb.WriteString(ven.Type.String())
	}

	return sb.String()
}

type ValueExprString struct {
	Token *Token
}

func (ves *ValueExprString) String() string {
	// Preserve the original quote style from the token
	switch ves.Token.Type {
	case CONST_STRING_DOUBLE_QUOTE:
		return "\"" + ves.Token.Lit + "\""
	case CONST_STRING_SINGLE_QUOTE:
		return "'" + ves.Token.Lit + "'"
	case CONST_STRING_BACKTICK_QOUTE:
		return "`" + ves.Token.Lit + "`"
	default:
		return ves.Token.Lit
	}
}

type ValueExprBool struct {
	Token *Token
}

func (veb *ValueExprBool) String() string {
	return veb.Token.Lit
}

type ValueExprNull struct {
	Token *Token
}

func (ven *ValueExprNull) String() string {
	return ven.Token.Lit
}

type DeclEnumSet struct {
	Name      *IdenExpr
	Value     Expr
	IsDefined bool
}

func (des *DeclEnumSet) String() string {
	var sb strings.Builder

	sb.WriteString(des.Name.String())
	if des.IsDefined {
		sb.WriteString(" = ")
		sb.WriteString(des.Value.String())
	}

	return sb.String()
}

type DeclEnum struct {
	Token      *Token // 'enum' token
	Name       *IdenExpr
	Values     []*DeclEnumSet
	CloseCurly *Token
}

func (de *DeclEnum) String() string {
	var sb strings.Builder

	sb.WriteString("enum ")
	sb.WriteString(de.Name.String())
	sb.WriteString(" {\n")
	for _, val := range de.Values {
		sb.WriteString("\t")
		sb.WriteString(val.String())
		sb.WriteString("\n")
	}
	sb.WriteString("}")

	return sb.String()
}

type DeclCustomType struct {
	Name *IdenExpr
}

func (dct *DeclCustomType) String() string {
	return dct.Name.String()
}

type DeclStringType struct {
	Name *IdenExpr
}

func (dst *DeclStringType) String() string {
	return dst.Name.String()
}

type DeclByteType struct {
	Name *IdenExpr
}

func (dbt *DeclByteType) String() string {
	return dbt.Name.String()
}

type DeclTimestampType struct {
	Name *IdenExpr
}

func (dtt *DeclTimestampType) String() string {
	return dtt.Name.String()
}

type DeclNumberType struct {
	Name *IdenExpr
}

func (dnt *DeclNumberType) String() string {
	return dnt.Name.String()
}

type DeclAnyType struct {
	Name *IdenExpr
}

func (dat *DeclAnyType) String() string {
	return dat.Name.String()
}

type DeclBoolType struct {
	Name *IdenExpr
}

func (dbt *DeclBoolType) String() string {
	return "bool"
}

type DeclArrayType struct {
	Token *Token
	Type  Decl
}

func (dat *DeclArrayType) String() string {
	return fmt.Sprintf("[]%s", dat.Type.String())
}

type DeclMapType struct {
	Token     *Token
	KeyType   Decl
	ValueType Decl
}

func (dmt *DeclMapType) String() string {
	return fmt.Sprintf("map<%s, %s>", dmt.KeyType.String(), dmt.ValueType.String())
}

type DeclModelField struct {
	Name    *IdenExpr
	Type    DeclType
	Optional bool
	Options []*AssignmentStmt
}

func (dmf *DeclModelField) String() string {
	if dmf.Optional {
		return fmt.Sprintf("%s?: %s", dmf.Name.String(), dmf.Type.String())
	}
	return fmt.Sprintf("%s: %s", dmf.Name.String(), dmf.Type.String())
}

type DeclModel struct {
	Token      *Token
	Name       *IdenExpr
	Extends    []*IdenExpr
	Fields     []*DeclModelField
	CloseCurly *Token
}

func (dm *DeclModel) String() string {
	var sb strings.Builder

	sb.WriteString("model ")
	sb.WriteString(dm.Name.String())
	sb.WriteString(" {\n")
	for _, field := range dm.Fields {
		sb.WriteString("\t")
		sb.WriteString(field.String())
		sb.WriteString("\n")
	}
	sb.WriteString("}")

	return sb.String()
}

type DeclNameTypePair struct {
	Name *IdenExpr
	Type DeclType
}

func (dntp *DeclNameTypePair) String() string {
	return dntp.Name.String() + ": " + dntp.Type.String()
}

type DeclServiceMethod struct {
	Name    *IdenExpr
	Args    []*DeclNameTypePair
	Returns []*DeclNameTypePair
	Options []*AssignmentStmt
}

func (dsm *DeclServiceMethod) String() string {
	var sb strings.Builder

	sb.WriteString(dsm.Name.String())
	sb.WriteString(" (")

	for i, arg := range dsm.Args {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(arg.String())
	}
	sb.WriteString(")")

	if len(dsm.Returns) > 0 {
		sb.WriteString(" => (")
		for i, ret := range dsm.Returns {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(ret.String())
		}
		sb.WriteString(")")
	}

	if len(dsm.Options) > 0 {
		sb.WriteString(" {\n")
		for _, opt := range dsm.Options {
			sb.WriteString("\t")
			sb.WriteString(opt.String())
			sb.WriteString("\n")
		}
		sb.WriteString("}")
	}

	return sb.String()
}

type DeclService struct {
	Token      *Token
	Name       *IdenExpr
	Methods    []*DeclServiceMethod
	CloseCurly *Token
}

func (ds *DeclService) String() string {
	var sb strings.Builder

	sb.WriteString("service ")
	sb.WriteString(ds.Name.String())
	sb.WriteString(" {\n")
	for _, method := range ds.Methods {
		sb.WriteString("\t")
		sb.WriteString(method.String())
		sb.WriteString("\n")
	}
	sb.WriteString("}")

	return sb.String()
}

type DeclError struct {
	Token      *Token
	Name       *IdenExpr
	Code       *ValueExprNumber
	Msg        *ValueExprString
	CloseCurly *Token
}

func (de *DeclError) String() string {
	var sb strings.Builder

	sb.WriteString("error ")
	sb.WriteString(de.Name.String())
	sb.WriteString(" { ")

	if de.Code != nil {
		sb.WriteString("Code = ")
		sb.WriteString(de.Code.String())
		sb.WriteString(" ")
	}

	sb.WriteString("Msg = ")
	sb.WriteString(de.Msg.String())
	sb.WriteString(" }")

	return sb.String()
}

type Program struct {
	Nodes    []Node
	Comments []*Token
}

// CommentedNode wraps a Node with its associated comments
type CommentedNode struct {
	LeadingComments []*Token // Comments on lines before this node
	Node            Node
	TrailingComment *Token // Comment on the same line after the node
}

func (p *Program) String() string {
	var sb strings.Builder

	for _, node := range p.Nodes {
		sb.WriteString(node.String())
		sb.WriteString("\n")
	}

	return sb.String()
}

func getTokenFromNode(n Node) *Token {
	switch node := n.(type) {
	case *IdenExpr:
		return node.Token
	case *ConstDecl:
		return node.Token
	case *ValueExprBool:
		return node.Token
	case *ValueExprNull:
		return node.Token
	case *ValueExprNumber:
		return node.Token
	case *ValueExprString:
		return node.Token
	case *DeclEnum:
		return node.Token
	case *DeclEnumSet:
		return node.Name.Token
	case *DeclModel:
		return node.Token
	case *DeclModelField:
		return node.Name.Token
	case *DeclCustomType:
		return node.Name.Token
	case *DeclStringType:
		return node.Name.Token
	case *DeclNumberType:
		return node.Name.Token
	case *DeclBoolType:
		return node.Name.Token
	case *DeclArrayType:
		return node.Token
	case *DeclMapType:
		return node.Token
	case *AssignmentStmt:
		return node.Name.Token
	case *DeclNameTypePair:
		return node.Name.Token
	case *DeclServiceMethod:
		return node.Name.Token
	case *DeclService:
		return node.Token
	case *DeclError:
		return node.Name.Token
	default:
		return nil
	}
}
