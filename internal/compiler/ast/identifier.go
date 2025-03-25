package ast

import (
	"strings"

	"ella.to/ella/internal/compiler/token"
)

//
// Identifier
//

type Identifier struct {
	Token *token.Token
}

var _ (Node) = (*Identifier)(nil)

func (i *Identifier) Format(sb *strings.Builder) {
	sb.WriteString(i.Token.Value)
}
