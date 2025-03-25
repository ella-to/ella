package ast

import (
	"strings"

	"ella.to/ella/internal/compiler/token"
)

//
// Value
//

type ValueBool struct {
	Token       *token.Token
	Value       bool
	UserDefined bool
}

var _ (Value) = (*ValueBool)(nil)

func (v *ValueBool) Format(sb *strings.Builder) {
	if v.Value {
		sb.WriteString("true")
	} else {
		sb.WriteString("false")
	}
}

func (v *ValueBool) value() {}

type ValueString struct {
	Token *token.Token
	Value string
}

var _ (Value) = (*ValueString)(nil)

func (v *ValueString) Format(sb *strings.Builder) {
	switch v.Token.Type {
	case token.ConstStringSingleQuote:
		sb.WriteString("'")
		sb.WriteString(v.Value)
		sb.WriteString("'")
	case token.ConstStringDoubleQuote:
		sb.WriteString("\"")
		sb.WriteString(v.Value)
		sb.WriteString("\"")
	case token.ConstStringBacktickQoute:
		sb.WriteString("`")
		sb.WriteString(v.Value)
		sb.WriteString("`")
	}
}

func (v *ValueString) value() {}

type ValueFloat struct {
	Token *token.Token
	Value float64
	Size  int // 32, 64
}

var _ (Value) = (*ValueFloat)(nil)

func (v *ValueFloat) Format(sb *strings.Builder) {
	sb.WriteString(v.Token.Value)
}

func (v *ValueFloat) value() {}

type ValueUint struct {
	Token *token.Token
	Value uint64
	Size  int // 8, 16, 32, 64
}

var _ (Value) = (*ValueUint)(nil)

func (v *ValueUint) Format(sb *strings.Builder) {
	sb.WriteString(v.Token.Value)
}

func (v *ValueUint) value() {}

type ValueInt struct {
	Token   *token.Token
	Value   int64
	Size    int  // 8, 16, 32, 64
	Defined bool // means if user explicitly set it
}

var _ (Value) = (*ValueInt)(nil)

func (v *ValueInt) Format(sb *strings.Builder) {
	sb.WriteString(v.Token.Value)
}

func (v *ValueInt) value() {}

type DurationScale int64

const (
	DurationScaleNanosecond  DurationScale = 1
	DurationScaleMicrosecond               = DurationScaleNanosecond * 1000
	DurationScaleMillisecond               = DurationScaleMicrosecond * 1000
	DurationScaleSecond                    = DurationScaleMillisecond * 1000
	DurationScaleMinute                    = DurationScaleSecond * 60
	DurationScaleHour                      = DurationScaleMinute * 60
)

func (d DurationScale) String() string {
	switch d {
	case DurationScaleNanosecond:
		return "ns"
	case DurationScaleMicrosecond:
		return "us"
	case DurationScaleMillisecond:
		return "ms"
	case DurationScaleSecond:
		return "s"
	case DurationScaleMinute:
		return "m"
	case DurationScaleHour:
		return "h"
	default:
		panic("unknown duration scale")
	}
}

type ValueDuration struct {
	Token *token.Token
	Value int64
	Scale DurationScale
}

var _ (Value) = (*ValueDuration)(nil)

func (v *ValueDuration) Format(sb *strings.Builder) {
	sb.WriteString(v.Token.Value)
}

func (v *ValueDuration) value() {}

type ByteSize int64

const (
	ByteSizeB  ByteSize = 1
	ByteSizeKB          = ByteSizeB * 1024
	ByteSizeMB          = ByteSizeKB * 1024
	ByteSizeGB          = ByteSizeMB * 1024
	ByteSizeTB          = ByteSizeGB * 1024
	ByteSizePB          = ByteSizeTB * 1024
	ByteSizeEB          = ByteSizePB * 1024
)

func (b ByteSize) String() string {
	switch b {
	case ByteSizeB:
		return "b"
	case ByteSizeKB:
		return "kb"
	case ByteSizeMB:
		return "mb"
	case ByteSizeGB:
		return "gb"
	case ByteSizeTB:
		return "tb"
	case ByteSizePB:
		return "pb"
	case ByteSizeEB:
		return "eb"
	default:
		panic("unknown byte size")
	}
}

type ValueByteSize struct {
	Token *token.Token
	Value int64
	Scale ByteSize
}

var _ (Value) = (*ValueByteSize)(nil)

func (v *ValueByteSize) Format(sb *strings.Builder) {
	sb.WriteString(v.Token.Value)
}

func (v *ValueByteSize) value() {}

type ValueNull struct {
	Token *token.Token
}

var _ Value = (*ValueNull)(nil)

func (v *ValueNull) Format(sb *strings.Builder) {
	sb.WriteString("null")
}

func (v *ValueNull) value() {}

type ValueVariable struct {
	Token *token.Token
}

var _ Value = (*ValueVariable)(nil)

func (v *ValueVariable) Format(sb *strings.Builder) {
	sb.WriteString(v.Token.Value)
}

func (v *ValueVariable) value() {}
