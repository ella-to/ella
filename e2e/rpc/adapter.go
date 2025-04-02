package rpc

import (
	"context"
	"io"
)

func NewRpcCallerMemory(handler Handler) Caller {
	return CallerFunc(func(ctx context.Context, req *Request) (body io.Reader, contentType string) {
		pr, pw := io.Pipe()
		go handler.Handle(ctx, req, pw)
		return pr, "application/json"
	})
}
