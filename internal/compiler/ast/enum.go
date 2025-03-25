package ast

import (
	"strings"

	"ella.to/ella/internal/compiler/token"
)

//
// Enum
//

type EnumSet struct {
	Name     *Identifier
	Value    *ValueInt
	Defined  bool
	Comments []*Comment
}

var _ (Expr) = (*EnumSet)(nil)

func (e *EnumSet) Format(sb *strings.Builder) {
	for _, comment := range e.Comments {
		sb.WriteString("    ")
		comment.Format(sb)
		sb.WriteString("\n")
	}

	sb.WriteString("    ")
	e.Name.Format(sb)
	if e.Value.Token != nil {
		sb.WriteString(" = ")
		e.Value.Format(sb)
	}
}

func (e *EnumSet) AddComments(comments ...*Comment) {
	e.Comments = append(e.Comments, comments...)
}

type Enum struct {
	Token    *token.Token
	Name     *Identifier
	Size     int // 8, 16, 32, 64 selected by compiler based on the largest and smallest values
	Sets     []*EnumSet
	Comments []*Comment
}

var _ (Expr) = (*Enum)(nil)

func (e *Enum) Format(sb *strings.Builder) {
	for _, comment := range e.Comments {
		if comment.Position != CommentTop {
			continue
		}
		comment.Format(sb)
		sb.WriteString("\n")
	}

	sb.WriteString("enum ")
	e.Name.Format(sb)
	sb.WriteString(" {\n")

	for i, set := range e.Sets {
		if i != 0 {
			sb.WriteString("\n")
		}

		set.Format(sb)
	}

	for _, comment := range e.Comments {
		if comment.Position != CommentBottom {
			continue
		}

		sb.WriteString("\n")
		sb.WriteString("    ")
		comment.Format(sb)
	}

	sb.WriteString("\n}")
}

func (e *Enum) AddComments(comments ...*Comment) {
	e.Comments = append(e.Comments, comments...)
}
