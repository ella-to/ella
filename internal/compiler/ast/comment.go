package ast

import (
	"strings"

	"ella.to/ella/internal/compiler/token"
)

//
// Comment
//

type CommentPosition int

const (
	CommentTop CommentPosition = iota
	CommentBottom
)

type Comment struct {
	Token    *token.Token
	Position CommentPosition
}

var _ (Node) = (*Comment)(nil)

func (c *Comment) Format(sb *strings.Builder) {
	sb.WriteString("# ")
	sb.WriteString(strings.TrimSpace(c.Token.Value))
}
