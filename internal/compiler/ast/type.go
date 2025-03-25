package ast

import (
	"strings"

	"ella.to/ella/internal/compiler/token"
)

//
// Type
//

type CustomType struct {
	Token *token.Token
}

var _ Type = (*CustomType)(nil)

func (c *CustomType) Format(sb *strings.Builder) {
	sb.WriteString(c.Token.Value)
}

func (c *CustomType) typ() {}

type Byte struct {
	Token *token.Token
}

var _ Type = (*Byte)(nil)

func (b *Byte) Format(sb *strings.Builder) {
	sb.WriteString("byte")
}

func (b *Byte) typ() {}

type Uint struct {
	Token *token.Token
	Size  int // 8, 16, 32, 64
}

var _ Type = (*Uint)(nil)

func (u *Uint) Format(sb *strings.Builder) {
	sb.WriteString(u.Token.Value)
}

func (u *Uint) typ() {}

type Int struct {
	Token *token.Token
	Size  int // 8, 16, 32, 64
}

var _ Type = (*Int)(nil)

func (u *Int) Format(sb *strings.Builder) {
	sb.WriteString(u.Token.Value)
}

func (u *Int) typ() {}

type Float struct {
	Token *token.Token
	Size  int // 32, 64
}

var _ Type = (*Float)(nil)

func (f *Float) Format(sb *strings.Builder) {
	sb.WriteString(f.Token.Value)
}

func (f *Float) typ() {}

type String struct {
	Token *token.Token
}

var _ Type = (*String)(nil)

func (s *String) Format(sb *strings.Builder) {
	sb.WriteString("string")
}

func (s *String) typ() {}

type Bool struct {
	Token *token.Token
}

var _ Type = (*Bool)(nil)

func (b *Bool) Format(sb *strings.Builder) {
	sb.WriteString("bool")
}

func (b *Bool) typ() {}

type Any struct {
	Token *token.Token
}

var _ Type = (*Any)(nil)

func (a *Any) Format(sb *strings.Builder) {
	sb.WriteString("any")
}

func (a *Any) typ() {}

type Array struct {
	Token *token.Token // this is the '[' token
	Type  Type
}

var _ Type = (*Array)(nil)

func (a *Array) Format(sb *strings.Builder) {
	sb.WriteString("[]")
	a.Type.Format(sb)
}

func (a *Array) typ() {}

type Map struct {
	Token *token.Token
	Key   Type
	Value Type
}

var _ Type = (*Map)(nil)

func (m *Map) Format(sb *strings.Builder) {
	sb.WriteString("map<")
	m.Key.Format(sb)
	sb.WriteString(", ")
	m.Value.Format(sb)
	sb.WriteString(">")
}

func (m *Map) typ() {}

type Timestamp struct {
	Token *token.Token
}

var _ Type = (*Timestamp)(nil)

func (t *Timestamp) Format(sb *strings.Builder) {
	sb.WriteString("timestamp")
}

func (t *Timestamp) typ() {}
