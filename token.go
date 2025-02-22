package main

type Token struct {
	Filename string
	Value    string
	Type     TokenType
	Start    int
	End      int
}

type TokenEmitter interface {
	EmitToken(token *Token)
}

type TokenEmitterFunc func(token *Token)

func (f TokenEmitterFunc) EmitToken(token *Token) {
	f(token)
}

type TokenIterator interface {
	NextToken() *Token
}

type TokenEmitterIterator struct {
	tokens chan *Token
	end    *Token
}

var _ TokenEmitter = (*TokenEmitterIterator)(nil)
var _ TokenIterator = (*TokenEmitterIterator)(nil)

func (e *TokenEmitterIterator) EmitToken(token *Token) {
	e.tokens <- token
}

func (e *TokenEmitterIterator) NextToken() *Token {
	tok, ok := <-e.tokens
	if !ok {
		return e.end
	} else if tok.Type == TokEOF {
		e.end = tok
		close(e.tokens)
		e.tokens = nil
	}

	return tok
}

func NewEmitterIterator() *TokenEmitterIterator {
	return &TokenEmitterIterator{
		tokens: make(chan *Token, 2),
	}
}

type TokenType int

const (
	TokError                    TokenType = -1   // Error token type which indicates error
	TokEOF                      TokenType = iota // EOF token type which indicates end of input
	TokIdentifier                                // identifier
	TokConst                                     // const
	TokEnum                                      // enum
	TokModel                                     // model
	TokHttp                                      // http
	TokRpc                                       // rpc
	TokService                                   // service
	TokByte                                      // byte
	TokBool                                      // bool
	TokInt8                                      // int8
	TokInt16                                     // int16
	TokInt32                                     // int32
	TokInt64                                     // int64
	TokUint8                                     // uint8
	TokUint16                                    // uint16
	TokUint32                                    // uint32
	TokUint64                                    // uint64
	TokFloat32                                   // float32
	TokFloat64                                   // float64
	TokTimestamp                                 // timestamp
	TokString                                    // string
	TokMap                                       // map
	TokArray                                     // array []
	TokAny                                       // any
	TokFile                                      // file
	TokStream                                    // stream
	TokConstDuration                             // 1ns, 1us, 1ms, 1s, 1m, 1h
	TokConstBytes                                // 1b, 1kb, 1mb, 1gb, 1tb, 1pb, 1eb
	TokConstFloat                                // 1.0
	TokConstInt                                  // 1
	TokConstStringSingleQuote                    // 'string'
	TokConstStringDoubleQuote                    // "string"
	TokConstStringBacktickQoute                  // `string`
	TokConstBool                                 // true, false
	TokConstNull                                 // null
	TokReturn                                    // =>
	TokAssign                                    // =
	TokOptional                                  // ?
	TokColon                                     // :
	TokComma                                     // ,
	TokExtend                                    // ...
	TokOpenCurly                                 // {
	TokCloseCurly                                // }
	TokOpenParen                                 // (
	TokCloseParen                                // )
	TokOpenAngle                                 // <
	TokCloseAngle                                // >
	TokComment                                   // # comment
	TokCustomError                               // error
)

func (tt TokenType) String() string {
	switch tt {
	case TokError:
		return "Error"
	case TokEOF:
		return "EOF"
	case TokIdentifier:
		return "Identifier"
	case TokConst:
		return "Const"
	case TokEnum:
		return "Enum"
	case TokModel:
		return "Model"
	case TokHttp:
		return "Http"
	case TokRpc:
		return "Rpc"
	case TokService:
		return "Service"
	case TokByte:
		return "Byte"
	case TokBool:
		return "Bool"
	case TokInt8:
		return "Int8"
	case TokInt16:
		return "Int16"
	case TokInt32:
		return "Int32"
	case TokInt64:
		return "Int64"
	case TokUint8:
		return "Uint8"
	case TokUint16:
		return "Uint16"
	case TokUint32:
		return "Uint32"
	case TokUint64:
		return "Uint64"
	case TokFloat32:
		return "Float32"
	case TokFloat64:
		return "Float64"
	case TokTimestamp:
		return "Timestamp"
	case TokString:
		return "String"
	case TokMap:
		return "Map"
	case TokArray:
		return "Array"
	case TokAny:
		return "Any"
	case TokFile:
		return "File"
	case TokStream:
		return "Stream"
	case TokConstDuration:
		return "ConstDuration"
	case TokConstBytes:
		return "ConstBytes"
	case TokConstFloat:
		return "ConstFloat"
	case TokConstInt:
		return "ConstInt"
	case TokConstStringSingleQuote:
		return "ConstStringSingleQuote"
	case TokConstStringDoubleQuote:
		return "ConstStringDoubleQuote"
	case TokConstStringBacktickQoute:
		return "ConstStringBacktickQoute"
	case TokConstBool:
		return "ConstBool"
	case TokConstNull:
		return "ConstNull"
	case TokReturn:
		return "Return"
	case TokAssign:
		return "Assign"
	case TokColon:
		return "Colon"
	case TokComma:
		return "Comma"
	case TokExtend:
		return "Extend"
	case TokOpenCurly:
		return "OpenCurly"
	case TokCloseCurly:
		return "CloseCurly"
	case TokOpenParen:
		return "OpenParen"
	case TokCloseParen:
		return "CloseParen"
	case TokOpenAngle:
		return "OpenAngle"
	case TokCloseAngle:
		return "CloseAngle"
	case TokComment:
		return "Comment"
	case TokCustomError:
		return "CustomError"
	default:
		return "Unknown"
	}
}
