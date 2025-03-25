package ast

import (
	"strings"

	"ella.to/ella/internal/compiler/token"
)

//
// Service
//

type Arg struct {
	Name   *Identifier
	Type   Type
	Stream bool
}

var _ (Node) = (*Arg)(nil)

func (a *Arg) Format(sb *strings.Builder) {
	a.Name.Format(sb)
	sb.WriteString(": ")
	if a.Stream {
		sb.WriteString("stream ")
	}
	a.Type.Format(sb)
}

type Return struct {
	Name   *Identifier
	Type   Type
	Stream bool
}

var _ (Node) = (*Return)(nil)

func (r *Return) Format(sb *strings.Builder) {
	r.Name.Format(sb)
	sb.WriteString(": ")
	if r.Stream {
		sb.WriteString("stream ")
	}
	r.Type.Format(sb)
}

type Method struct {
	Name     *Identifier
	Args     []*Arg
	Returns  []*Return
	Options  *Options
	Comments []*Comment
}

var _ (Expr) = (*Method)(nil)

func (m *Method) Format(sb *strings.Builder) {
	for _, comment := range m.Comments {
		sb.WriteString("\n    ")
		comment.Format(sb)
	}

	sb.WriteString("\n    ")

	m.Name.Format(sb)
	sb.WriteString(" (")

	for i, arg := range m.Args {
		if i != 0 {
			sb.WriteString(", ")
		}
		arg.Format(sb)
	}

	sb.WriteString(")")

	if len(m.Returns) > 0 {
		sb.WriteString(" => (")
		for i, ret := range m.Returns {
			if i != 0 {
				sb.WriteString(", ")
			}
			ret.Format(sb)
		}
		sb.WriteString(")")
	}

	if len(m.Options.List) > 0 || len(m.Options.Comments) > 0 {
		m.Options.Format(sb)
	}
}

func (m *Method) AddComments(comments ...*Comment) {
	m.Comments = append(m.Comments, comments...)
}

type ServiceType int

const (
	_           ServiceType = iota
	ServiceRPC              // rpc
	ServiceHTTP             // http
)

func (m ServiceType) String() string {
	switch m {
	case ServiceRPC:
		return "rpc"
	case ServiceHTTP:
		return "http"
	default:
		return "unknown"
	}
}

type Service struct {
	Token    *token.Token
	Name     *Identifier
	Type     ServiceType
	Methods  []*Method
	Comments []*Comment
}

var _ (Expr) = (*Service)(nil)

func (s *Service) Format(sb *strings.Builder) {
	for _, comment := range s.Comments {
		if comment.Position != CommentTop {
			continue
		}
		comment.Format(sb)
		sb.WriteString("\n")
	}

	sb.WriteString("service ")
	s.Name.Format(sb)
	sb.WriteString(" {")

	for _, method := range s.Methods {
		method.Format(sb)
	}

	for _, comment := range s.Comments {
		if comment.Position != CommentBottom {
			continue
		}

		sb.WriteString("\n    ")
		comment.Format(sb)
	}

	sb.WriteString("\n}")
}

func (s *Service) AddComments(comments ...*Comment) {
	s.Comments = append(s.Comments, comments...)
}
