package token

type Token struct {
	Filename string
	Value    string
	Type     Type
	Start    int
	End      int
}

type Emitter interface {
	EmitToken(token *Token)
}

type EmitterFunc func(token *Token)

func (f EmitterFunc) EmitToken(token *Token) {
	f(token)
}

type Iterator interface {
	NextToken() *Token
}

type EmitterIterator struct {
	tokens chan *Token
	end    *Token
}

var (
	_ Emitter  = (*EmitterIterator)(nil)
	_ Iterator = (*EmitterIterator)(nil)
)

func (e *EmitterIterator) EmitToken(token *Token) {
	e.tokens <- token
}

func (e *EmitterIterator) NextToken() *Token {
	tok, ok := <-e.tokens
	if !ok {
		return e.end
	} else if tok.Type == EOF {
		e.end = tok
		close(e.tokens)
		e.tokens = nil
	}

	return tok
}

func NewEmitterIterator() *EmitterIterator {
	return &EmitterIterator{
		tokens: make(chan *Token, 2),
	}
}
