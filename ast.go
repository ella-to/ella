package main

import (
	"strconv"
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

//
// Comment
//

type CommentPosition int

const (
	CommentTop CommentPosition = iota
	CommentBottom
)

type Comment struct {
	Token    *Token
	Position CommentPosition
}

var _ (Node) = (*Comment)(nil)

func (c *Comment) Format(sb *strings.Builder) {
	sb.WriteString("# ")
	sb.WriteString(strings.TrimSpace(c.Token.Value))
}

//
// Value
//

type ValueBool struct {
	Token       *Token
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
	Token *Token
	Value string
}

var _ (Value) = (*ValueString)(nil)

func (v *ValueString) Format(sb *strings.Builder) {
	switch v.Token.Type {
	case TokConstStringSingleQuote:
		sb.WriteString("'")
		sb.WriteString(v.Value)
		sb.WriteString("'")
	case TokConstStringDoubleQuote:
		sb.WriteString("\"")
		sb.WriteString(v.Value)
		sb.WriteString("\"")
	case TokConstStringBacktickQoute:
		sb.WriteString("`")
		sb.WriteString(v.Value)
		sb.WriteString("`")
	}
}

func (v *ValueString) value() {}

type ValueFloat struct {
	Token *Token
	Value float64
	Size  int // 32, 64
}

var _ (Value) = (*ValueFloat)(nil)

func (v *ValueFloat) Format(sb *strings.Builder) {
	sb.WriteString(v.Token.Value)
}

func (v *ValueFloat) value() {}

type ValueUint struct {
	Token *Token
	Value uint64
	Size  int // 8, 16, 32, 64
}

var _ (Value) = (*ValueUint)(nil)

func (v *ValueUint) Format(sb *strings.Builder) {
	sb.WriteString(v.Token.Value)
}

func (v *ValueUint) value() {}

type ValueInt struct {
	Token   *Token
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
	Token *Token
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
	Token *Token
	Value int64
	Scale ByteSize
}

var _ (Value) = (*ValueByteSize)(nil)

func (v *ValueByteSize) Format(sb *strings.Builder) {
	sb.WriteString(v.Token.Value)
}

func (v *ValueByteSize) value() {}

type ValueNull struct {
	Token *Token
}

var _ Value = (*ValueNull)(nil)

func (v *ValueNull) Format(sb *strings.Builder) {
	sb.WriteString("null")
}

func (v *ValueNull) value() {}

type ValueVariable struct {
	Token *Token
}

var _ Value = (*ValueVariable)(nil)

func (v *ValueVariable) Format(sb *strings.Builder) {
	sb.WriteString(v.Token.Value)
}

func (v *ValueVariable) value() {}

//
// Type
//

type File struct {
	Token *Token
}

var _ Type = (*File)(nil)

func (f *File) Format(sb *strings.Builder) {
	sb.WriteString("file")
}

func (f *File) typ() {}

type CustomType struct {
	Token *Token
}

var _ Type = (*CustomType)(nil)

func (c *CustomType) Format(sb *strings.Builder) {
	sb.WriteString(c.Token.Value)
}

func (c *CustomType) typ() {}

type Byte struct {
	Token *Token
}

var _ Type = (*Byte)(nil)

func (b *Byte) Format(sb *strings.Builder) {
	sb.WriteString("byte")
}

func (b *Byte) typ() {}

type Uint struct {
	Token *Token
	Size  int // 8, 16, 32, 64
}

var _ Type = (*Uint)(nil)

func (u *Uint) Format(sb *strings.Builder) {
	sb.WriteString(u.Token.Value)
}

func (u *Uint) typ() {}

type Int struct {
	Token *Token
	Size  int // 8, 16, 32, 64
}

var _ Type = (*Int)(nil)

func (u *Int) Format(sb *strings.Builder) {
	sb.WriteString(u.Token.Value)
}

func (u *Int) typ() {}

type Float struct {
	Token *Token
	Size  int // 32, 64
}

var _ Type = (*Float)(nil)

func (f *Float) Format(sb *strings.Builder) {
	sb.WriteString(f.Token.Value)
}

func (f *Float) typ() {}

type String struct {
	Token *Token
}

var _ Type = (*String)(nil)

func (s *String) Format(sb *strings.Builder) {
	sb.WriteString("string")
}

func (s *String) typ() {}

type Bool struct {
	Token *Token
}

var _ Type = (*Bool)(nil)

func (b *Bool) Format(sb *strings.Builder) {
	sb.WriteString("bool")
}

func (b *Bool) typ() {}

type Any struct {
	Token *Token
}

var _ Type = (*Any)(nil)

func (a *Any) Format(sb *strings.Builder) {
	sb.WriteString("any")
}

func (a *Any) typ() {}

type Array struct {
	Token *Token // this is the '[' token
	Type  Type
}

var _ Type = (*Array)(nil)

func (a *Array) Format(sb *strings.Builder) {
	sb.WriteString("[]")
	a.Type.Format(sb)
}

func (a *Array) typ() {}

type Map struct {
	Token *Token
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
	Token *Token
}

var _ Type = (*Timestamp)(nil)

func (t *Timestamp) Format(sb *strings.Builder) {
	sb.WriteString("timestamp")
}

func (t *Timestamp) typ() {}

//
// Identifier
//

type Identifier struct {
	Token *Token
}

var _ (Node) = (*Identifier)(nil)

func (i *Identifier) Format(sb *strings.Builder) {
	sb.WriteString(i.Token.Value)
}

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

//
// Const
//

type Const struct {
	Token      *Token
	Identifier *Identifier
	Value      Value
	Comments   []*Comment
}

var _ (Expr) = (*Const)(nil)

func (c *Const) Format(sb *strings.Builder) {
	for _, comment := range c.Comments {
		comment.Format(sb)
		sb.WriteString("\n")
	}

	sb.WriteString("const ")
	c.Identifier.Format(sb)
	sb.WriteString(" = ")
	c.Value.Format(sb)
}

func (c *Const) AddComments(comments ...*Comment) {
	c.Comments = append(c.Comments, comments...)
}

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
	Token    *Token
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
	Token    *Token
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

//
// Service
//

type Arg struct {
	Name *Identifier
	Type Type
}

var _ (Node) = (*Arg)(nil)

func (a *Arg) Format(sb *strings.Builder) {
	a.Name.Format(sb)
	sb.WriteString(": ")
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

type MethodType int

const (
	_             MethodType = iota
	MethodRPC                // rpc
	MethodHTTP               // http
	MethodRpcHttp            // rpc, http
)

func (m MethodType) String() string {
	switch m {
	case MethodRPC:
		return "rpc"
	case MethodHTTP:
		return "http"
	case MethodRpcHttp:
		return "rpc,http"
	default:
		return "unknown"
	}
}

type Method struct {
	Type     MethodType // rpc, http
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

	switch m.Type {
	case MethodRPC:
		sb.WriteString("rpc ")
	case MethodHTTP:
		sb.WriteString("http ")
	case MethodRpcHttp:
		sb.WriteString("rpc, http ")
	}

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

type Service struct {
	Token    *Token
	Name     *Identifier
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

//
// Custom Error
//

type CustomError struct {
	Token      *Token
	Name       *Identifier
	Code       int64
	HttpStatus int
	Msg        *ValueString
	Comments   []*Comment
}

var _ (Expr) = (*CustomError)(nil)

func (c *CustomError) Format(sb *strings.Builder) {
	for _, comment := range c.Comments {
		sb.WriteString("\n")
		comment.Format(sb)
	}

	if len(c.Comments) > 0 {
		sb.WriteString("\n")
	}
	sb.WriteString("error ")
	c.Name.Format(sb)
	sb.WriteString(" { ")

	if c.Code != 0 {
		sb.WriteString("Code = ")
		sb.WriteString(strconv.FormatInt(c.Code, 10))
		sb.WriteString(" ")
	}

	if c.HttpStatus != 0 {
		sb.WriteString("HttpStatus = ")
		sb.WriteString(HttpStatusCode2String[c.HttpStatus])
		sb.WriteString(" ")
	}

	sb.WriteString("Msg = ")
	c.Msg.Format(sb)
	sb.WriteString(" }")
}

func (c *CustomError) AddComments(comments ...*Comment) {
	c.Comments = append(c.Comments, comments...)
}

var HttpStatusCode2String = map[int]string{
	100: "Continue",
	101: "SwitchingProtocols",
	102: "Processing",
	103: "EarlyHints",
	200: "OK",
	201: "Created",
	202: "Accepted",
	203: "NonAuthoritativeInfo",
	204: "NoContent",
	205: "ResetContent",
	206: "PartialContent",
	207: "MultiStatus",
	208: "AlreadyReported",
	226: "IMUsed",
	300: "MultipleChoices",
	301: "MovedPermanently",
	302: "Found",
	303: "SeeOther",
	304: "NotModified",
	305: "UseProxy",
	307: "TemporaryRedirect",
	308: "PermanentRedirect",
	400: "BadRequest",
	401: "Unauthorized",
	402: "PaymentRequired",
	403: "Forbidden",
	404: "NotFound",
	405: "MethodNotAllowed",
	406: "NotAcceptable",
	407: "ProxyAuthRequired",
	408: "RequestTimeout",
	409: "Conflict",
	410: "Gone",
	411: "LengthRequired",
	412: "PreconditionFailed",
	413: "RequestEntityTooLarge",
	414: "RequestURITooLong",
	415: "UnsupportedMediaType",
	416: "RequestedRangeNotSatisfiable",
	417: "ExpectationFailed",
	418: "Teapot",
	421: "MisdirectedRequest",
	422: "UnprocessableEntity",
	423: "Locked",
	424: "FailedDependency",
	425: "TooEarly",
	426: "UpgradeRequired",
	428: "PreconditionRequired",
	429: "TooManyRequests",
	431: "RequestHeaderFieldsTooLarge",
	451: "UnavailableForLegalReasons",
	500: "InternalServerError",
	501: "NotImplemented",
	502: "BadGateway",
	503: "ServiceUnavailable",
	504: "GatewayTimeout",
	505: "HTTPVersionNotSupported",
	506: "VariantAlsoNegotiates",
	507: "InsufficientStorage",
	508: "LoopDetected",
	510: "NotExtended",
	511: "NetworkAuthenticationRequired",
}

var HttpStatusString2Code = map[string]int{
	"Continue":                      100,
	"SwitchingProtocols":            101,
	"Processing":                    102,
	"EarlyHints":                    103,
	"OK":                            200,
	"Created":                       201,
	"Accepted":                      202,
	"NonAuthoritativeInfo":          203,
	"NoContent":                     204,
	"ResetContent":                  205,
	"PartialContent":                206,
	"MultiStatus":                   207,
	"AlreadyReported":               208,
	"IMUsed":                        226,
	"MultipleChoices":               300,
	"MovedPermanently":              301,
	"Found":                         302,
	"SeeOther":                      303,
	"NotModified":                   304,
	"UseProxy":                      305,
	"TemporaryRedirect":             307,
	"PermanentRedirect":             308,
	"BadRequest":                    400,
	"Unauthorized":                  401,
	"PaymentRequired":               402,
	"Forbidden":                     403,
	"NotFound":                      404,
	"MethodNotAllowed":              405,
	"NotAcceptable":                 406,
	"ProxyAuthRequired":             407,
	"RequestTimeout":                408,
	"Conflict":                      409,
	"Gone":                          410,
	"LengthRequired":                411,
	"PreconditionFailed":            412,
	"RequestEntityTooLarge":         413,
	"RequestURITooLong":             414,
	"UnsupportedMediaType":          415,
	"RequestedRangeNotSatisfiable":  416,
	"ExpectationFailed":             417,
	"Teapot":                        418,
	"MisdirectedRequest":            421,
	"UnprocessableEntity":           422,
	"Locked":                        423,
	"FailedDependency":              424,
	"TooEarly":                      425,
	"UpgradeRequired":               426,
	"PreconditionRequired":          428,
	"TooManyRequests":               429,
	"RequestHeaderFieldsTooLarge":   431,
	"UnavailableForLegalReasons":    451,
	"InternalServerError":           500,
	"NotImplemented":                501,
	"BadGateway":                    502,
	"ServiceUnavailable":            503,
	"GatewayTimeout":                504,
	"HTTPVersionNotSupported":       505,
	"VariantAlsoNegotiates":         506,
	"InsufficientStorage":           507,
	"LoopDetected":                  508,
	"NotExtended":                   510,
	"NetworkAuthenticationRequired": 511,
}
