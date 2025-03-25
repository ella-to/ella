package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"ella.to/sse"
)

/* RPC (JSON)

Request
- body
	{}

Response
- body
	{}

*/

/* HTTP (JSON, multipart/form-data, octet-stream, event-stream)

2 possible ways to send a request:
- JSON
- multipart/form-data

3 possible ways to receive a response:
- octet-stream
- event-stream
- JSON

- JSON -> JSON (GENERATED)
- JSON -> octet-stream
- JSON -> event-stream
- multipart/form-data -> JSON (GENERATED)
- multipart/form-data -> octet-stream
- multipart/form-data -> event-stream

-[x] handleJsonToJson (Generated)
-[x] handleJsonToBinary
-[x] handleJsonToSSE
-[x] handleMultipartToJson (Generated)
-[x] handleMultipartToBinary
-[x] handleMultipartToSSE

Request
- Content-Type: application/json
- body
	{
		"id": "<id>",
		"method": "method",
		"params": []
	}

- Content-Type: multipart/form-data
- body
	Field: id
	Value: <id>

	Field: method
	Value: <method>

	Field: params
	Value: {}

	Field: file
	Value: <raw binary data>

	Field: file
	Value: <raw binary data>

	...

Response
- Content-Type: application/octet-stream
- body
	<raw binary data>

- Content-Type: text/event-stream
- body
	id: 1
	event: message
	data: {}

	id: 2
	event: error
	data: { "code": 0, "message": "error message", "cause": "error cause" }

	id: 3
	event: done
	data:

- Content-Type: application/json
- body
	{
		"result": {}
	}

	{
		"error": {
			"code": 0,
			"message": "error message"
			"cause": "error cause"
		}
	}

*/

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Cause   error  `json:"cause"`
}

var _ error = (*Error)(nil)

func (e *Error) Error() string {
	return e.Message
}

type Request struct {
	Id          string            `json:"id"`
	Method      string            `json:"method"`
	Params      json.RawMessage   `json:"params"`
	ContentType string            `json:"-"`
	Files       []*multipart.Part `json:"-"`
	Boundary    string            `json:"-"`
}

type Sender interface {
	Send(ctx context.Context, req *Request, resp io.Writer)
}

type SenderFunc func(ctx context.Context, req *Request, resp io.Writer)

var _ Sender = (*SenderFunc)(nil)

func (f SenderFunc) Send(ctx context.Context, req *Request, resp io.Writer) {
	f(ctx, req, resp)
}

type Registererer interface {
	Register(name string, sender Sender)
}

//

type Registry struct {
	senders map[string]Sender
}

var (
	_ Registererer = (*Registry)(nil)
	_ Sender       = (*Registry)(nil)
)

func (r *Registry) Register(name string, sender Sender) {
	r.senders[name] = sender
}

func (r *Registry) Send(ctx context.Context, req *Request, resp io.Writer) {
	sender, ok := r.senders[req.Method]
	if !ok {
		writeError(resp, &Error{
			Code:    0,
			Message: "method not found",
		})
		return
	}
	sender.Send(ctx, req, resp)
}

func parseParams[A any](r io.Reader) (a A, err error) {
	err = json.NewDecoder(r).Decode(&a)
	return
}

func handleJsonToJson0[A any](fn func(context.Context, A) error) Sender {
	return SenderFunc(func(ctx context.Context, req *Request, resp io.Writer) {
		params, err := parseParams[A](bytes.NewReader(req.Params))
		if err != nil {
			writeError(resp, err)
			return
		}

		writeResults(resp)(fn(ctx, params))
	})
}

func handleJsonToBinary[A any](fn func(context.Context, A) (io.Reader, error)) Sender {
	return SenderFunc(func(ctx context.Context, req *Request, resp io.Writer) {
		params, err := parseParams[A](bytes.NewReader(req.Params))
		if err != nil {
			writeError(resp, err)
			return
		}

		r, err := fn(ctx, params)
		if err != nil {
			writeError(resp, err)
			return
		}

		if w, ok := resp.(http.ResponseWriter); ok {
			w.WriteHeader(http.StatusOK)
			// TODO: change the content type later
			w.Header().Set("Content-Type", "application/octet-stream")
		}

		io.Copy(resp, r)
	})
}

func handleJsonToStream[A, R any](fn func(context.Context, A) (<-chan R, error)) Sender {
	return SenderFunc(func(ctx context.Context, req *Request, resp io.Writer) {
		var err error

		params, err := parseParams[A](bytes.NewReader(req.Params))
		if err != nil {
			writeError(resp, err)
			return
		}

		ch, err := fn(ctx, params)
		if err != nil {
			writeError(resp, err)
			return
		}

		streamEvents(ch, resp)
	})
}

func handleMultipartToJson0[A any](fn func(context.Context, A, func() (string, io.Reader, error)) error) Sender {
	return SenderFunc(func(ctx context.Context, req *Request, resp io.Writer) {
		params, err := parseParams[A](bytes.NewReader(req.Params))
		if err != nil {
			writeError(resp, err)
			return
		}

		idx := 0
		writeResults(resp)(fn(ctx, params, func() (string, io.Reader, error) {
			if idx >= len(req.Files) {
				return "", nil, io.EOF
			}

			part := req.Files[idx]
			idx++

			return part.FileName(), part, nil
		}))
	})
}

func handleMultipartToBinary[A any](fn func(context.Context, A, func() (string, io.Reader, error)) (io.Reader, error)) Sender {
	return SenderFunc(func(ctx context.Context, req *Request, resp io.Writer) {
		params, err := parseParams[A](bytes.NewReader(req.Params))
		if err != nil {
			writeError(resp, err)
			return
		}

		idx := 0
		r, err := fn(ctx, params, func() (string, io.Reader, error) {
			if idx >= len(req.Files) {
				return "", nil, io.EOF
			}

			part := req.Files[idx]
			idx++

			return part.FileName(), part, nil
		})
		if err != nil {
			writeError(resp, err)
			return
		}

		if w, ok := resp.(http.ResponseWriter); ok {
			w.WriteHeader(http.StatusOK)
			// TODO: change the content type later
			w.Header().Set("Content-Type", "application/octet-stream")
		}

		io.Copy(resp, r)
	})
}

func handleMultipartToStream[A, R any](fn func(context.Context, A, func() (string, io.Reader, error)) (<-chan R, error)) Sender {
	return SenderFunc(func(ctx context.Context, req *Request, resp io.Writer) {
		var err error

		params, err := parseParams[A](bytes.NewReader(req.Params))
		if err != nil {
			return
		}

		idx := 0
		ch, err := fn(ctx, params, func() (string, io.Reader, error) {
			if idx >= len(req.Files) {
				return "", nil, io.EOF
			}

			part := req.Files[idx]
			idx++

			return part.FileName(), part, nil
		})
		if err != nil {
			return
		}

		streamEvents(ch, resp)
	})
}

func streamEvents[T any](ch <-chan T, resp io.Writer) {
	var id int64
	pusher, err := sse.NewPusher(resp, 500*time.Millisecond)
	if err != nil {
		writeError(resp, err)
		return
	}

	defer func() {
		pusher.Close()
		id++

		if err != nil {
			var sb strings.Builder
			writeError(&sb, err)
			pusher.Push(sse.NewMessage(fmt.Sprintf("%d", id), "error", sb.String()))
			id++
		}

		pusher.Push(sse.NewMessage(fmt.Sprintf("%d", id), "end", ""))
	}()

	var buffer bytes.Buffer

	for e := range ch {
		id++
		buffer.Reset()
		if err := json.NewEncoder(&buffer).Encode(e); err != nil {
			return
		}

		if err := pusher.Push(sse.NewMessage(fmt.Sprintf("%d", id), "data", buffer.String())); err != nil {
			return
		}
	}
}

func writeError(resp io.Writer, err error) {
	switch e := err.(type) {
	case *Error:
		{
			json.NewEncoder(resp).Encode(e)
		}
	default:
		writeError(resp, &Error{
			Code:    0,
			Message: "something unknown happens",
			Cause:   err,
		})
	}
}

func writeResults(out io.Writer) func(...any) {
	return func(rets ...any) {
		w, isHttpWriter := out.(http.ResponseWriter)

		if isHttpWriter {
			w.Header().Set("Content-Type", "application/json")
		}

		if len(rets) > 0 && rets[len(rets)-1] != nil {
			if isHttpWriter {
				w.WriteHeader(http.StatusExpectationFailed)
			}
			writeError(out, rets[len(rets)-1].(error))
		}

		if isHttpWriter {
			w.WriteHeader(http.StatusOK)
		}

		json.NewEncoder(out).Encode(struct {
			Result any `json:"result"`
		}{
			Result: rets,
		})
	}
}

func NewHttpHandler(srv Sender) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := parseRequest(r.Body, r.Header.Get("Content-Type"))
		if err != nil {
			writeError(w, err)
			return
		}

		srv.Send(r.Context(), req, w)
	})
}

func parseRequest(r io.Reader, contentType string) (*Request, error) {
	req := new(Request)

	switch contentType {
	case "application/json":
		{
			if err := json.NewDecoder(r).Decode(req); err != nil {
				return nil, err
			}
		}
	case "multipart/form-data":
		{
			boundary := strings.TrimPrefix(contentType, "multipart/form-data; boundary=")
			if boundary == "" {
				return nil, errors.New("missing boundary")
			}
			req.Boundary = boundary

			reader := multipart.NewReader(r, boundary)

			for {
				part, err := reader.NextPart()
				if errors.Is(err, io.EOF) {
					break
				} else if err != nil {
					return nil, err
				}

				switch part.FormName() {
				case "id":
					{
						data, err := io.ReadAll(part)
						if err != nil {
							return nil, fmt.Errorf("failed to read multipart id: %w", err)
						}
						req.Id = string(data)
					}
				case "method":
					{
						data, err := io.ReadAll(part)
						if err != nil {
							return nil, fmt.Errorf("failed to read multipart method: %w", err)
						}
						req.Method = string(data)
					}
				case "params":
					{
						data, err := io.ReadAll(part)
						if err != nil {
							return nil, fmt.Errorf("failed to read multipart params: %w", err)
						}
						req.Params = data
					}
				case "file":
					{
						req.Files = append(req.Files, part)
					}
				default:
					{
						return nil, fmt.Errorf("unknown multipart field: %s", part.FormName())
					}
				}
			}
		}
	}

	req.ContentType = contentType

	return req, nil
}

func main() {
}
