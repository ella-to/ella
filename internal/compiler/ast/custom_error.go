package ast

import (
	"strconv"
	"strings"

	"ella.to/ella/internal/compiler/token"
)

//
// Custom Error
//

type CustomError struct {
	Token    *token.Token
	Name     *Identifier
	Code     int64
	Msg      *ValueString
	Comments []*Comment
}

var _ (Expr) = (*CustomError)(nil)

func (c *CustomError) Format(sb *strings.Builder) {
	for _, comment := range c.Comments {
		sb.WriteString("\n")
		comment.Format(sb)
	}

	if len(c.Comments) > 0 {
		sb.WriteString("\n")
	}
	sb.WriteString("error ")
	c.Name.Format(sb)
	sb.WriteString(" { ")

	if c.Code != 0 {
		sb.WriteString("Code = ")
		sb.WriteString(strconv.FormatInt(c.Code, 10))
		sb.WriteString(" ")
	}

	sb.WriteString("Msg = ")
	c.Msg.Format(sb)
	sb.WriteString(" }")
}

func (c *CustomError) AddComments(comments ...*Comment) {
	c.Comments = append(c.Comments, comments...)
}
