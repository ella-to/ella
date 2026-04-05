package compiler

import (
	"bufio"
	"io"
	"strings"
)

type Pos struct {
	Offset int
	Line   int
	Column int
	Src    string
}

type Scanner struct {
	rs *RuneScanner
}

func NewScanner(r io.Reader, src string) *Scanner {
	return &Scanner{
		rs: NewRuneScanner(r, src),
	}
}

func (s *Scanner) Scan() (*Token, error) {
	var lit string
	var pos Pos

	for {
		s.rs.AcceptRun(" \t\n\r") // skip whitespace
		s.rs.CleanBuffer()

		ch := s.rs.Peek()

		switch ch {
		case -1:
			return newToken(EOF, s.rs.pos, ""), nil
		case '#':
			lit, pos, _ = s.rs.AcceptRunUntil("\n\r")
			return newToken(COMMENT, pos, lit), nil
		case '?':
			ch, pos = s.rs.Next()
			return newToken(OPTIONAL, pos, string(ch)), nil
		case ':':
			ch, pos = s.rs.Next()
			return newToken(COLON, pos, string(ch)), nil
		case ',':
			ch, pos = s.rs.Next()
			return newToken(COMMA, pos, string(ch)), nil
		case '{':
			ch, pos = s.rs.Next()
			return newToken(OPEN_CURLY, pos, string(ch)), nil
		case '}':
			ch, pos = s.rs.Next()
			return newToken(CLOSE_CURLY, pos, string(ch)), nil
		case '(':
			ch, pos = s.rs.Next()
			return newToken(OPEN_PAREN, pos, string(ch)), nil
		case ')':
			ch, pos = s.rs.Next()
			return newToken(CLOSE_PAREN, pos, string(ch)), nil
		case '<':
			ch, pos = s.rs.Next()
			return newToken(OPEN_ANGLE, pos, string(ch)), nil
		case '>':
			ch, pos = s.rs.Next()
			return newToken(CLOSE_ANGLE, pos, string(ch)), nil
		case '[':
			ch, pos = s.rs.Next()
			return newToken(OPEN_SQURE, pos, string(ch)), nil
		case ']':
			ch, pos = s.rs.Next()
			return newToken(CLOSE_SQURE, pos, string(ch)), nil
		case '=':
			ch, pos = s.rs.Next()
			return newToken(EQUAL, pos, string(ch)), nil
		case '.':
			ch, pos = s.rs.Next()
			return newToken(DOT, pos, string(ch)), nil
		case '\'':
			_, pos = s.rs.Next()
			lit, _, _ = s.rs.AcceptRunUntil("'\n\r")
			if ch = s.rs.Peek(); ch != '\'' {
				return nil, NewError(newToken(ERROR, pos, lit), "unclosed single quote string")
			}
			s.rs.Next()
			return newToken(CONST_STRING_SINGLE_QUOTE, pos, lit), nil
		case '"':
			_, pos = s.rs.Next()
			lit, _, _ = s.rs.AcceptRunUntil("\"\n\r")
			if ch = s.rs.Peek(); ch != '"' {
				return nil, NewError(newToken(ERROR, pos, lit), "unclosed double quote string")
			}
			s.rs.Next()
			return newToken(CONST_STRING_DOUBLE_QUOTE, pos, lit), nil
		case '`':
			_, pos = s.rs.Next()
			lit, _, _ = s.rs.AcceptRunUntil("`")
			if ch = s.rs.Peek(); ch != '`' {
				return nil, NewError(newToken(ERROR, pos, lit), "unclosed backtick quote string")
			}
			s.rs.Next()
			return newToken(CONST_STRING_BACKTICK_QOUTE, pos, lit), nil
		default:
			var tok *Token
			var err error
			// Only attempt numeric scanning when the token can actually begin with a number.
			if strings.ContainsRune("+-0123456789", ch) {
				tok, err = s.ScanNumber()
				if err != nil {
					return nil, err
				}
				if tok.Type != UNKNOWN {
					s.rs.CleanBuffer()
					return tok, nil
				}
			}

			tok = s.ScanReservedWord()
			s.rs.CleanBuffer()
			if tok.Type != UNKNOWN {
				return tok, nil
			}

			return newToken(IDENTIFIER, tok.Pos, tok.Lit), nil
		}
	}
}

func (s *Scanner) ScanReservedWord() *Token {
	lit, pos, ok := s.rs.AcceptRunUntil("=,.:?{}()<>[]# \t\n\r")
	if !ok {
		return newToken(ERROR, pos, "unable to scan token")
	}

	switch lit {
	case "const":
		return newToken(CONST, pos, lit)
	case "enum":
		return newToken(ENUM, pos, lit)
	case "model":
		return newToken(MODEL, pos, lit)
	case "service":
		return newToken(SERVICE, pos, lit)
	case "error":
		return newToken(CUSTOM_ERROR, pos, lit)
	case "byte":
		return newToken(BYTE, pos, lit)
	case "bool":
		return newToken(BOOL, pos, lit)
	case "int8":
		return newToken(INT8, pos, lit)
	case "int16":
		return newToken(INT16, pos, lit)
	case "int32":
		return newToken(INT32, pos, lit)
	case "int64":
		return newToken(INT64, pos, lit)
	case "uint8":
		return newToken(UINT8, pos, lit)
	case "uint16":
		return newToken(UINT16, pos, lit)
	case "uint32":
		return newToken(UINT32, pos, lit)
	case "uint64":
		return newToken(UINT64, pos, lit)
	case "float32":
		return newToken(FLOAT32, pos, lit)
	case "float64":
		return newToken(FLOAT64, pos, lit)
	case "timestamp":
		return newToken(TIMESTAMP, pos, lit)
	case "string":
		return newToken(STRING, pos, lit)
	case "any":
		return newToken(ANY, pos, lit)
	case "map":
		return newToken(MAP, pos, lit)
	default:
		return newToken(UNKNOWN, pos, lit)
	}
}

func (s *Scanner) ScanNumber() (*Token, error) {
	var emptyPos Pos
	var pos Pos

	_, startPos, ok := s.rs.Accept("+-")
	if ok {
		pos = startPos
	}

	digits := "0123456789"

	{
		var startsWithZero bool

		_, startPos, startsWithZero = s.rs.Accept("0")
		if startsWithZero {
			if pos == emptyPos {
				pos = startPos
			}
			// Only check for hex prefix if we started with 0
			_, _, isHex := s.rs.Accept("xX")
			if isHex {
				digits = "0123456789abcdefABCDEF"
			}
		}
	}

	digits += "_"

	_, startPos, ok = s.rs.AcceptRun(digits)
	if ok {
		if pos == emptyPos {
			pos = startPos
		}
	}

	content := s.rs.Buffer()
	if len(content) == 0 || strings.HasPrefix(content, "_") {
		return newToken(UNKNOWN, startPos, ""), nil
	}

	_, _, ok = s.rs.Accept(".")
	if ok {
		_, startPos, ok = s.rs.AcceptRun(digits)
		if !ok {
			return nil, NewError(newToken(ERROR, startPos, ""), "expected digit after decimal point")
		}
	}

	if _, _, ok = s.rs.Accept("eE"); ok {
		s.rs.Accept("+-")
		s.rs.AcceptRun("0123456789_")
	}

	if strings.HasSuffix(s.rs.Buffer(), "_") {
		return nil, NewError(newToken(ERROR, pos, s.rs.Buffer()), "number cannot end with underscore")
	}

	return newToken(CONST_NUMBER, pos, s.rs.Buffer()), nil
}

type RuneScanner struct {
	rr            io.RuneReader
	pos           Pos
	ch            rune
	hasBackup     bool
	addedToBuffer bool
	buffer        []rune
}

func (r *RuneScanner) BufferLen() int {
	return len(r.buffer)
}

func (r *RuneScanner) Buffer() string {
	return string(r.buffer)
}

func (r *RuneScanner) CleanBuffer() {
	r.buffer = r.buffer[:0]
}

func (r *RuneScanner) Accept(valid string) (rune, Pos, bool) {
	ch, pos := r.Next()
	if strings.ContainsRune(valid, ch) {
		return ch, pos, true
	}
	if ch != -1 {
		r.Backup()
	}
	return ch, pos, false
}

func (r *RuneScanner) AcceptRun(valid string) (string, Pos, bool) {
	var sb strings.Builder
	var pos Pos
	var ok bool
	var ch rune
	var nextPos Pos

	for {
		ch, nextPos = r.Next()
		if !strings.ContainsRune(valid, ch) {
			break
		}

		if !ok {
			pos = nextPos
			ok = true
		}

		sb.WriteRune(ch)
	}

	if ch != -1 {
		r.Backup()
	}

	return sb.String(), pos, ok
}

func (r *RuneScanner) AcceptRunUntil(invalid string) (string, Pos, bool) {
	var sb strings.Builder
	var pos Pos
	var ok bool
	var ch rune
	var nextPos Pos

	for {
		ch, nextPos = r.Next()
		if ch == -1 || strings.ContainsRune(invalid, ch) {
			break
		}

		if !ok {
			pos = nextPos
			ok = true
		}

		sb.WriteRune(ch)
	}

	if ch != -1 {
		r.Backup()
	}

	return sb.String(), pos, ok
}

func (r *RuneScanner) Next() (rune, Pos) {
	if r.hasBackup {
		r.hasBackup = false
		r.addedToBuffer = true
		r.buffer = append(r.buffer, r.ch)
		return r.ch, r.pos
	}

	var err error
	r.ch, _, err = r.rr.ReadRune()
	if err != nil {
		r.ch = -1
		return r.ch, r.pos
	}

	r.pos.Offset++
	if r.ch == '\n' {
		r.pos.Line++
		r.pos.Column = 0
	} else {
		r.pos.Column++
	}

	r.addedToBuffer = true
	r.buffer = append(r.buffer, r.ch)

	return r.ch, r.pos
}

func (r *RuneScanner) Peek() rune {
	if !r.hasBackup {
		ch, _ := r.Next()
		if ch != -1 {
			r.Backup()
		}
	}
	return r.ch
}

func (r *RuneScanner) Backup() {
	if r.addedToBuffer {
		r.buffer = r.buffer[:len(r.buffer)-1]
		r.addedToBuffer = false
	}

	r.hasBackup = true
}

func NewRuneScanner(r io.Reader, src string) *RuneScanner {
	return &RuneScanner{
		rr: bufio.NewReader(r),
		pos: Pos{
			Offset: -1,
			Line:   1,
			Column: 0,
			Src:    src,
		},
	}
}
