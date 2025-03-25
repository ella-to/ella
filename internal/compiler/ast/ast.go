package ast

import (
	"strings"
)

type Node interface {
	Format(buffer *strings.Builder)
}

type Expr interface {
	Node
	AddComments(comments ...*Comment)
}

type Type interface {
	Node
	typ()
}

type Value interface {
	Node
	value()
}
