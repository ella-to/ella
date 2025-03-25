package ast

import "strings"

//
// Option
//

type Option struct {
	Name     *Identifier
	Value    Value
	Comments []*Comment
}

var _ (Expr) = (*Option)(nil)

func (o *Option) Format(sb *strings.Builder) {
	for _, comment := range o.Comments {
		sb.WriteString("\n        ")
		comment.Format(sb)
	}

	sb.WriteString("\n        ")
	o.Name.Format(sb)
	if v, ok := o.Value.(*ValueBool); ok {
		// it means that it's just a flag option without value
		// so we don't need to print the value
		if v.Token == nil {
			return
		}
	}

	sb.WriteString(" = ")
	o.Value.Format(sb)
}

func (o *Option) AddComments(comments ...*Comment) {
	o.Comments = append(o.Comments, comments...)
}

type Options struct {
	List     []*Option
	Comments []*Comment
}

var _ (Expr) = (*Options)(nil)

func (o *Options) Format(sb *strings.Builder) {
	sb.WriteString(" {")
	for _, option := range o.List {
		option.Format(sb)
	}

	for _, comment := range o.Comments {
		sb.WriteString("\n        ")
		comment.Format(sb)
	}

	sb.WriteString("\n    }")
}

func (o *Options) AddComments(comments ...*Comment) {
	o.Comments = append(o.Comments, comments...)
}
