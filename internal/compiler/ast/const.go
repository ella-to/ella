package ast

import (
	"strings"

	"ella.to/ella/internal/compiler/token"
)

//
// Const
//

type Const struct {
	Token      *token.Token
	Identifier *Identifier
	Value      Value
	Comments   []*Comment
}

var _ (Expr) = (*Const)(nil)

func (c *Const) Format(sb *strings.Builder) {
	for _, comment := range c.Comments {
		comment.Format(sb)
		sb.WriteString("\n")
	}

	sb.WriteString("const ")
	c.Identifier.Format(sb)
	sb.WriteString(" = ")
	c.Value.Format(sb)
}

func (c *Const) AddComments(comments ...*Comment) {
	c.Comments = append(c.Comments, comments...)
}
