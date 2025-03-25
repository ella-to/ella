package ast

import (
	"strings"

	"ella.to/ella/internal/compiler/token"
)

//
// Model
//

type Field struct {
	Name       *Identifier
	Type       Type
	IsOptional bool
	Options    *Options
	Comments   []*Comment
}

var _ (Expr) = (*Field)(nil)

func (f *Field) Format(sb *strings.Builder) {
	for i, comment := range f.Comments {
		if i != 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("    ")
		comment.Format(sb)
	}

	if len(f.Comments) > 0 {
		sb.WriteString("\n")
	}

	sb.WriteString("    ")
	f.Name.Format(sb)
	if f.IsOptional {
		sb.WriteString("?")
	}
	sb.WriteString(": ")
	f.Type.Format(sb)

	if len(f.Options.List) == 0 && len(f.Options.Comments) == 0 {
		return
	}

	f.Options.Format(sb)
}

func (f *Field) AddComments(comments ...*Comment) {
	f.Comments = append(f.Comments, comments...)
}

type Extend struct {
	Name     *Identifier
	Comments []*Comment
}

var _ (Expr) = (*Extend)(nil)

func (e *Extend) Format(sb *strings.Builder) {
	for _, comment := range e.Comments {
		sb.WriteString("\n    ")
		comment.Format(sb)
	}

	sb.WriteString("    ...")
	e.Name.Format(sb)
}

func (e *Extend) AddComments(comments ...*Comment) {
	e.Comments = append(e.Comments, comments...)
}

type Model struct {
	Token    *token.Token
	Name     *Identifier
	Extends  []*Extend
	Fields   []*Field
	Comments []*Comment
}

var _ (Expr) = (*Model)(nil)

func (m *Model) Format(sb *strings.Builder) {
	for _, comment := range m.Comments {
		if comment.Position != CommentTop {
			continue
		}
		comment.Format(sb)
		sb.WriteString("\n")
	}

	sb.WriteString("model ")
	m.Name.Format(sb)

	sb.WriteString(" {")

	for _, extend := range m.Extends {
		sb.WriteString("\n")
		extend.Format(sb)
	}

	for _, field := range m.Fields {
		sb.WriteString("\n")
		field.Format(sb)
	}

	for _, comment := range m.Comments {
		if comment.Position != CommentBottom {
			continue
		}

		sb.WriteString("\n    ")
		comment.Format(sb)
	}

	sb.WriteString("\n}")
}

func (m *Model) AddComments(comments ...*Comment) {
	m.Comments = append(m.Comments, comments...)
}
