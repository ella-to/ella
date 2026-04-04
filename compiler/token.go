package compiler

type Token struct {
	Type TokenType
	Pos  Pos
	Lit  string
}

func (t *Token) IsInjected() bool {
	return t.Pos.Line == -1 && t.Pos.Column == -1 && t.Pos.Offset == -1
}

func newToken(typ TokenType, pos Pos, lit string) *Token {
	return &Token{
		Type: typ,
		Pos:  pos,
		Lit:  lit,
	}
}

func newInjectedToken(typ TokenType, lit string) *Token {
	return &Token{
		Type: typ,
		Pos: Pos{
			Offset: -1,
			Line:   -1,
			Column: -1,
		},
		Lit: lit,
	}
}

type TokenType int

const (
	EOF TokenType = iota
	UNKNOWN
	ERROR
	COMMENT
	IDENTIFIER
	CONST
	ENUM
	MODEL
	SERVICE
	BYTE
	BOOL
	INT8
	INT16
	INT32
	INT64
	UINT8
	UINT16
	UINT32
	UINT64
	FLOAT32
	FLOAT64
	TIMESTAMP
	STRING
	ANY
	MAP
	CONST_NUMBER
	CONST_STRING_SINGLE_QUOTE
	CONST_STRING_DOUBLE_QUOTE
	CONST_STRING_BACKTICK_QOUTE
	CONST_BOOL
	CONST_NULL
	EQUAL
	OPTIONAL
	COLON
	COMMA
	DOT
	OPEN_CURLY
	CLOSE_CURLY
	OPEN_PAREN
	CLOSE_PAREN
	OPEN_ANGLE
	CLOSE_ANGLE
	OPEN_SQURE
	CLOSE_SQURE
	CUSTOM_ERROR
)

func (tt TokenType) String() string {
	return tokenTypes[tt]
}

var tokenTypes = [...]string{
	ERROR:                       "ERROR",
	UNKNOWN:                     "UNKNOWN",
	EOF:                         "EOF",
	COMMENT:                     "COMMENT",
	IDENTIFIER:                  "IDENTIFIER",
	CONST:                       "CONST",
	ENUM:                        "ENUM",
	MODEL:                       "MODEL",
	SERVICE:                     "SERVICE",
	BYTE:                        "BYTE",
	BOOL:                        "BOOL",
	INT8:                        "INT8",
	INT16:                       "INT16",
	INT32:                       "INT32",
	INT64:                       "INT64",
	UINT8:                       "UINT8",
	UINT16:                      "UINT16",
	UINT32:                      "UINT32",
	UINT64:                      "UINT64",
	FLOAT32:                     "FLOAT32",
	FLOAT64:                     "FLOAT64",
	TIMESTAMP:                   "TIMESTAMP",
	STRING:                      "STRING",
	ANY:                         "ANY",
	MAP:                         "MAP",
	CONST_NUMBER:                "CONST_NUMBER",
	CONST_STRING_SINGLE_QUOTE:   "CONST_STRING_SINGLE_QUOTE",
	CONST_STRING_DOUBLE_QUOTE:   "CONST_STRING_DOUBLE_QUOTE",
	CONST_STRING_BACKTICK_QOUTE: "CONST_STRING_BACKTICK_QOUTE",
	CONST_BOOL:                  "CONST_BOOL",
	CONST_NULL:                  "CONST_NULL",
	EQUAL:                       "EQUAL",
	OPTIONAL:                    "OPTIONAL",
	COLON:                       "COLON",
	COMMA:                       "COMMA",
	DOT:                         "DOT",
	OPEN_CURLY:                  "OPEN_CURLY",
	CLOSE_CURLY:                 "CLOSE_CURLY",
	OPEN_PAREN:                  "OPEN_PAREN",
	CLOSE_PAREN:                 "CLOSE_PAREN",
	OPEN_ANGLE:                  "OPEN_ANGLE",
	CLOSE_ANGLE:                 "CLOSE_ANGLE",
	OPEN_SQURE:                  "OPEN_SQURE",
	CLOSE_SQURE:                 "CLOSE_SQURE",
	CUSTOM_ERROR:                "CUSTOM_ERROR",
}
