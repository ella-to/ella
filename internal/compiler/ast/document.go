package ast

import "strings"

//
// Document
//

type Document struct {
	Comments []*Comment
	Consts   []*Const
	Enums    []*Enum
	Models   []*Model
	Services []*Service
	Errors   []*CustomError
}

var _ (Expr) = (*Document)(nil)

func (d *Document) Format(sb *strings.Builder) {
	// Consts
	//
	for i, c := range d.Consts {
		if i != 0 {
			sb.WriteString("\n")
		}
		c.Format(sb)
	}

	if len(d.Consts) > 0 && (len(d.Enums) > 0 || len(d.Models) > 0 || len(d.Services) > 0 || len(d.Errors) > 0) {
		sb.WriteString("\n\n")
	}

	// Enums
	//

	for i, e := range d.Enums {
		if i != 0 {
			sb.WriteString("\n\n")
		}

		e.Format(sb)
	}

	if len(d.Enums) > 0 && (len(d.Models) > 0 || len(d.Services) > 0 || len(d.Errors) > 0) {
		sb.WriteString("\n\n")
	}

	// Models
	//

	for i, m := range d.Models {
		if i != 0 {
			sb.WriteString("\n\n")
		}

		m.Format(sb)
	}

	if len(d.Models) > 0 && (len(d.Services) > 0 || len(d.Errors) > 0) {
		sb.WriteString("\n\n")
	}

	// Services
	//

	for i, s := range d.Services {
		if i != 0 {
			sb.WriteString("\n")
		}

		s.Format(sb)
	}

	if len(d.Services) > 0 && len(d.Errors) > 0 {
		sb.WriteString("\n\n")
	}

	// Errors

	for i, e := range d.Errors {
		if i != 0 {
			sb.WriteString("\n")
		}

		e.Format(sb)
	}

	// Comments (Remaining)
	neededNewline := (len(d.Consts) > 0 || len(d.Enums) > 0 || len(d.Services) > 0 || len(d.Errors) > 0) && len(d.Comments) > 0

	if neededNewline {
		sb.WriteString("\n")
	}

	for i, comment := range d.Comments {
		if i != 0 {
			sb.WriteString("\n")
		}
		comment.Format(sb)
	}
}

func (d *Document) AddComments(comments ...*Comment) {
	d.Comments = append(d.Comments, comments...)
}
