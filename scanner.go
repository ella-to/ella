package main

import "strings"

func Lex(l *Lexer) State {
	ignoreWhiteSpace(l)

	switch l.Peek() {
	case 0:
		l.Emit(TokEOF)
		return nil
	case '=':
		l.Next()
		if l.Peek() == '>' {
			l.Next()
			l.Emit(TokReturn)
			return Lex
		}
		l.Emit(TokAssign)
		return Lex
	case ':':
		l.Next()
		l.Emit(TokColon)
		return Lex
	case ',':
		l.Next()
		l.Emit(TokComma)
		return Lex
	case '.':
		l.Next()
		if l.Next() != '.' {
			l.Errorf("extend requires 3 consecutive dots")
			return nil
		}
		if l.Next() != '.' {
			l.Errorf("extend requires 3 consecutive dots")
			return nil
		}
		l.Emit(TokExtend)
		return Lex
	case '{':
		l.Next()
		l.Emit(TokOpenCurly)
		return Lex
	case '}':
		l.Next()
		l.Emit(TokCloseCurly)
		return Lex
	case '(':
		l.Next()
		l.Emit(TokOpenParen)
		return Lex
	case ')':
		l.Next()
		l.Emit(TokCloseParen)
		return Lex
	case '<':
		l.Next()
		l.Emit(TokOpenAngle)
		return Lex
	case '>':
		l.Next()
		l.Emit(TokCloseAngle)
		return Lex
	case '[':
		l.Next()
		if l.Peek() != ']' {
			l.Errorf("expect ] to close array")
			return nil
		}
		l.Next()
		l.Emit(TokArray)
		return Lex
	case '#':
		l.Next()
		l.Ignore()
		l.AcceptRunUntil("\n\r")
		l.Emit(TokComment)
	case '\'':
		l.Next()
		l.Ignore()
		l.AcceptRunUntil("'\n\r")
		if l.Peek() != '\'' {
			l.Errorf("expect ' to close single quote")
			return nil
		}
		l.Emit(TokConstStringSingleQuote)
		l.Next()
		l.Ignore()
	case '"':
		l.Next()
		l.Ignore()
		l.AcceptRunUntil("\"\n\r")
		if l.Peek() != '"' {
			l.Errorf("expect \" to close double quote")
			return nil
		}
		l.Emit(TokConstStringDoubleQuote)
		l.Next()
		l.Ignore()
	case '`':
		l.Next()
		l.Ignore()
		l.AcceptRunUntil("`")
		if l.Peek() != '`' {
			l.Errorf("expect ` to close back multi line quote")
			return nil
		}
		l.Emit(TokConstStringBacktickQoute)
		l.Next()
		l.Ignore()
	default:
		ok, found := parseNumber(l)
		if found {
			return Lex
		} else if !ok {
			return nil
		}

		l.AcceptRunUntil("=,.:?{}()<>[]# \t\n\r")
		if l.Current() == "" {
			l.Errorf("expect something but got nothing")
			return nil
		}
		if !reservedKeywrod(l) {
			l.Emit(TokIdentifier)
		}
	}

	return Lex
}

func Number(l *Lexer) State {
	parseNumber(l)
	return nil
}

func parseNumber(l *Lexer) (ok bool, found bool) {
	isFloat := false

	l.Accept("+-")

	digits := "0123456789"
	if l.Accept("0") && l.Accept("xX") {
		digits = "0123456789abcdefABCDEF"
	}

	digits += "_"

	l.AcceptRun(digits)

	if len(l.Current()) == 0 || strings.HasPrefix(l.Current(), "_") {
		return true, false // not founding number but no error
	}

	if l.Accept(".") {
		isFloat = true
		if !l.AcceptRun(digits) {
			l.Errorf("expected digit after decimal point")
			return false, false // not founding number and with error
		}
	}

	if l.Accept("eE") {
		l.Accept("+-")
		l.AcceptRun("0123456789_")
	}

	if strings.HasSuffix(l.Current(), "_") {
		l.Errorf("expected digit after each underscore")
		return false, false // not founding number and with error
	}

	l.Accept("i")

	isDuration := false
	isBytes := isBytesTypeNum(l)
	if !isBytes && isDurationTypeNum(l) {
		isDuration = true
	}

	peek := l.Peek()

	if peek == 0 || peek == ' ' || peek == '\t' || peek == '\n' || peek == '\r' || peek == '#' {
		if strings.Contains(l.Current(), "__") {
			l.Errorf("expected digit after each underscore")
			return false, false // not founding number and with error
		}

		if isFloat && isBytes {
			l.Errorf("bytes number can't be presented as float")
			return false, false
		} else if isFloat && isDuration {
			l.Errorf("duration number can't be presented as float")
			return false, false
		} else if isFloat {
			l.Emit(TokConstFloat)
		} else if isBytes {
			l.Emit(TokConstBytes)
		} else if isDuration {
			l.Emit(TokConstDuration)
		} else {
			l.Emit(TokConstInt)
		}

		return true, true // founding number and no error
	}

	l.Errorf("unexpected character after number: %c", peek)

	return false, false // not founding number and with error
}

// checking if there is any B, KB, MB, GB, TB, PB, EB, ZB, YB
func isBytesTypeNum(l *Lexer) bool {
	isBytes := false
	if l.Accept("b") {
		isBytes = true
	} else {
		value := l.PeekN(2)
		if value == "kb" || // kilobyte
			value == "mb" || // megabyte
			value == "gb" || // gigabyte
			value == "tb" || // terabyte
			value == "pb" || // petabyte
			value == "eb" { // exabyte
			isBytes = true
			l.Next()
			l.Next()
		}
	}
	return isBytes
}

// checking if there is any ms, s, m, h, which represent millisecond, second, minute, hour
func isDurationTypeNum(l *Lexer) bool {
	value := l.PeekN(2)

	if value == "ns" || value == "us" || value == "ms" { // microsecond
		l.Next()
		l.Next()
		return true
	} else {
		return l.Accept("smh")
	}
}

func reservedKeywrod(l *Lexer) bool {
	switch l.Current() {
	case "const":
		l.Emit(TokConst)
		return true
	case "enum":
		l.Emit(TokEnum)
		return true
	case "model":
		l.Emit(TokModel)
		return true
	case "http":
		l.Emit(TokHttp)
		return true
	case "rpc":
		l.Emit(TokRpc)
		return true
	case "service":
		l.Emit(TokService)
		return true
	case "byte":
		l.Emit(TokByte)
		return true
	case "bool":
		l.Emit(TokBool)
		return true
	case "int8":
		l.Emit(TokInt8)
		return true
	case "int16":
		l.Emit(TokInt16)
		return true
	case "int32":
		l.Emit(TokInt32)
		return true
	case "int64":
		l.Emit(TokInt64)
		return true
	case "uint8":
		l.Emit(TokUint8)
		return true
	case "uint16":
		l.Emit(TokUint16)
		return true
	case "uint32":
		l.Emit(TokUint32)
		return true
	case "uint64":
		l.Emit(TokUint64)
		return true
	case "float32":
		l.Emit(TokFloat32)
		return true
	case "float64":
		l.Emit(TokFloat64)
		return true
	case "timestamp":
		l.Emit(TokTimestamp)
		return true
	case "string":
		l.Emit(TokString)
		return true
	case "map":
		l.Emit(TokMap)
		return true
	case "any":
		l.Emit(TokAny)
		return true
	case "file":
		l.Emit(TokFile)
		return true
	case "stream":
		l.Emit(TokStream)
		return true
	case "true", "false":
		l.Emit(TokConstBool)
		return true
	case "null":
		l.Emit(TokConstNull)
		return true
	case "error":
		l.Emit(TokCustomError)
		return true
	default:
		return false
	}
}

func ignoreWhiteSpace(l *Lexer) (newLine bool) {
	l.AcceptRun(" \t\r\n")
	if strings.Contains(l.Current(), "\n") {
		newLine = true
	}
	l.Ignore()
	return newLine
}
