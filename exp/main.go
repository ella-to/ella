package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type request struct {
	Id     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type response struct {
	Id      string `json:"id"`
	Results []any  `json:"results"`
}

type Error struct {
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
	Cause error  `json:"cause,omitempty"`
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("(%d): %s: %s", e.Code, e.Msg, e.Cause)
	}
	return fmt.Sprintf("(%d): %s", e.Code, e.Msg)
}

func (e *Error) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, e)
}

type MathService interface {
	Add(ctx context.Context, a, b int) (int, error)
	Download(ctx context.Context, id string) (io.Reader, error)
}

func CreateMathServiceFuncLookup(svc MathService) CallerFuncLookup {
	return func(method string) (CallerFunc, bool) {
		switch method {
		case "Add":
			{
				type Args struct {
					A int `json:"a"`
					B int `json:"b"`
				}

				return caller2(func(ctx context.Context, args Args) (int, error) {
					return svc.Add(ctx, args.A, args.B)
				}), true
			}
		case "Download":
			{
				type Args struct {
					Id string `json:"id"`
				}

				return caller2(func(ctx context.Context, args Args) (int, error) {
					return svc.Download(ctx, args.Id)
				}), true
			}
		}
		return nil, false
	}
}

func HttpHandler(serviceFinder ServiceFinder) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		segments := strings.Split(req.Method, ".")
		if len(segments) != 2 {
			http.Error(w, "invalid method", http.StatusBadRequest)
			return
		}

		service := segments[0]
		method := segments[1]

		caller, ok := serviceFinder.FindService(service)
		if !ok {
			http.Error(w, "service not found", http.StatusNotFound)
			return
		}

		callerFunc, ok := caller(method)
		if !ok {
			http.Error(w, "method not found", http.StatusNotFound)
			return
		}

		io.Copy(w, bytes.NewReader(callerFunc(r.Context(), req.Params)))
	})
}

type ServiceFinder interface {
	FindService(name string) (CallerFuncLookup, bool)
}

type MethodFinder interface {
	FindMethod(name string) (CallerFunc, bool)
}

type MethodCaller interface {
	CallMethod(ctx context.Context, in io.Reader) (out io.Reader)
}

type CallerFuncLookup func(method string) (CallerFunc, bool)
type CallerFunc func(ctx context.Context, in io.Reader) (out io.Reader)

func downloadCaller[A any](fn func(context.Context, A) (io.Reader, error)) func(context.Context, io.Reader) io.Reader {
	return func(ctx context.Context, in io.Reader) io.Reader {
		var a A
		if err := json.NewDecoder(in).Decode(&a); err != nil {
			return marshalErrorReader(err)
		}

		r, err := fn(ctx, a)
		if err != nil {
			return marshalErrorReader(err)
		}

		return r
	}
}

func channelCaller[A, B any](fn func(context.Context, A) (<-chan B, error)) func(context.Context, io.Reader, io.Writer) {
	return func(ctx context.Context, in io.Reader, out io.Writer) {
		var a A
		if err := json.NewDecoder(in).Decode(&a); err != nil {
			return marshalErrorReader(err)
		}

		ch, err := fn(ctx, a)
		if err != nil {
			return marshalErrorReader(err)
		}

		_ = ch

		return nil
	}
}

func caller[A any](fn func(context.Context, A) error) CallerFunc {
	return func(ctx context.Context, in []byte) (out []byte) {
		var a A
		if err := json.Unmarshal(in, &a); err != nil {
			return marshalError(err)
		}
		return marshalArray(fn(ctx, a))
	}
}

func caller2[A, R1 any](fn func(context.Context, A) (R1, error)) CallerFunc {
	return func(ctx context.Context, in []byte) (out []byte) {
		var a A
		if err := json.Unmarshal(in, &a); err != nil {
			return marshalError(err)
		}
		return marshalArray(fn(ctx, a))
	}
}

func caller3[A, R1, R2 any](fn func(context.Context, A) (R1, R2, error)) CallerFunc {
	return func(ctx context.Context, in []byte) (out []byte) {
		var a A
		if err := json.Unmarshal(in, &a); err != nil {
			return marshalError(err)
		}
		return marshalArray(fn(ctx, a))
	}
}

func marshalError(err error) json.RawMessage {
	b, _ := json.Marshal(err)
	return b
}

func marshalErrorReader(err error) io.Reader {
	return bytes.NewReader(marshalError(err))
}

// last element in the array is error, if it's nil, it's a success
func marshalArray(args ...any) json.RawMessage {
	if len(args) > 0 && args[len(args)-1] != nil {
		return marshalError(args[len(args)-1].(error))
	}
	b, _ := json.Marshal(args)
	return b
}

func main() {

}
