package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"ella.to/sse"
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

//
// Receiver
//

type Receiver interface {
	Receive(ctx context.Context, in io.Reader, out io.Writer)
}

type ReceiverFunc func(ctx context.Context, in io.Reader, out io.Writer)

func (f ReceiverFunc) Receive(ctx context.Context, in io.Reader, out io.Writer) {
	f(ctx, in, out)
}

//
// Dialer
//

type Dialer interface {
	Dial(ctx context.Context, in io.Reader, out io.Writer)
}

type DialerFunc func(ctx context.Context, in io.Reader, out io.Writer)

func (f DialerFunc) Dial(ctx context.Context, in io.Reader, out io.Writer) {
	f(ctx, in, out)
}

//
// Context Key
//

type ctxKey string

const (
	ctxMetadataKey = ctxKey("ella:metadata")
)

type Metadata struct {
	ContentType      string
	FormDataBoundary string
	Id               string
	Method           string
	Params           json.RawMessage
	Files            []*multipart.Part
}

func MetaDataFromContext(ctx context.Context) *Metadata {
	meta, _ := ctx.Value(ctxMetadataKey).(*Metadata)
	return meta
}

// NewHttpReceiver support only the following accept types
// - application/json (request/response json encoding)
// - text/event-stream (server-sent events)
// - application/octet-stream (raw binary data)
// - multipart/form-data (file uploads)
func NewHttpReceiver(recv Receiver) http.Handler {
	var supportedTypes = []string{
		"application/json",
		"text/event-stream",
		"application/octet-stream",
		"multipart/form-data",
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			http.Error(w, "missing Content-Type header", http.StatusBadRequest)
			return
		}

		// quick and dirty check
		var supported bool
		for _, t := range supportedTypes {
			if strings.Contains(contentType, t) {
				supported = true
				break
			}
		}
		if !supported {
			http.Error(w, "unsupported Content-Type header", http.StatusNotAcceptable)
			return
		}

		meta := &Metadata{}

		ctx := context.WithValue(r.Context(), ctxMetadataKey, meta)

		switch contentType {
		case "application/json":
			{
				req := new(request)
				if err := json.NewDecoder(r.Body).Decode(req); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}

				meta.Id = req.Id
				meta.Method = req.Method
				meta.Params = req.Params
			}
		case "multipart/form-data":
			{
				boundary := strings.TrimPrefix(contentType, "multipart/form-data; boundary=")
				if boundary == "" {
					http.Error(w, "missing boundary in Content-Type header", http.StatusBadRequest)
					return
				}
				meta.FormDataBoundary = boundary

				reader, err := r.MultipartReader()
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}

				for {
					part, err := reader.NextPart()
					if errors.Is(err, io.EOF) {
						break
					} else if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					switch part.FormName() {
					case "id":
						{
							data, err := io.ReadAll(part)
							if err != nil {
								http.Error(w, err.Error(), http.StatusInternalServerError)
								return
							}

							meta.Id = string(data)
						}
					case "method":
						{
							data, err := io.ReadAll(part)
							if err != nil {
								http.Error(w, err.Error(), http.StatusInternalServerError)
								return
							}

							meta.Method = string(data)
						}
					case "params":
						{
							data, err := io.ReadAll(part)
							if err != nil {
								http.Error(w, err.Error(), http.StatusInternalServerError)
								return
							}

							meta.Params = data
						}
					case "file":
						{
							meta.Files = append(meta.Files, part)
						}
					default:
						{
							http.Error(w, "unknown form field", http.StatusBadRequest)
							return
						}
					}
				}
			}
		}

		recv.Receive(ctx, r.Body, w)
	})
}

type Registererer interface {
	Register(name string, recv Receiver)
}

type Registry struct {
	mapper map[string]Receiver
}

var _ Receiver = (*Registry)(nil)
var _ Registererer = (*Registry)(nil)

func (r *Registry) Receive(ctx context.Context, in io.Reader, out io.Writer) {
	meta := MetaDataFromContext(ctx)
	if meta == nil {
		return
	}

	recv, ok := r.mapper[meta.Method]
	if !ok {
		return
	}

	recv.Receive(ctx, in, out)
}

func (r *Registry) Register(name string, recv Receiver) {
	if _, ok := r.mapper[name]; ok {
		panic(fmt.Sprintf("receiver %q already registered", name))
	}
	r.mapper[name] = recv
}

func CreateRegistry() *Registry {
	return &Registry{
		mapper: make(map[string]Receiver),
	}
}

func NewHttpDialer(url string, client *http.Client) Dialer {
	if client == nil {
		client = http.DefaultClient
	}

	return DialerFunc(func(ctx context.Context, in io.Reader, out io.Writer) {
		meta := MetaDataFromContext(ctx)

		r, err := http.NewRequestWithContext(ctx, http.MethodPost, url, in)
		if err != nil {
			return
		}

		r.Header.Set("Content-Type", meta.ContentType)

		resp, err := client.Do(r)
		if err != nil {
			return
		}

		io.Copy(out, resp.Body)
	})
}

type httpUserServiceClient struct {
}

var _ HttpUserService = (*httpUserServiceClient)(nil)

func CreateHttpUserServiceClient(dialer Dialer) HttpUserService {
}

func errorAsReader(err error) io.Reader {
	return nil
}

type Event struct {
	Id string `json:"id"`
}

type File struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type TestService interface {
	Add(ctx context.Context, a, b int) (int, error)
	Events(ctx context.Context, id string) (<-chan *Event, error)
	Download(ctx context.Context, id string) (io.Reader, error)
	Upload(ctx context.Context, id string, files func() (string, io.Reader, error)) ([]*File, error)
}

func RegisterRpcTestService(r *Registry, srv TestService) {
	r.Register(
		"RpcTestService.Add",
		processAndReturn2(
			func(
				ctx context.Context,
				args struct {
					A int `json:"a"`
					B int `json:"b"`
				},
			) (int, error) {
				return srv.Add(ctx, args.A, args.B)
			},
		),
	)

	r.Register(
		"RpcTestService.Events",
		processStream(
			func(
				ctx context.Context,
				args struct {
					Id string `json:"id"`
				},
			) (
				<-chan *Event,
				error,
			) {
				return srv.Events(ctx, args.Id)
			},
		),
	)

	r.Register(
		"RpcTestService.Download",
		processBinaryStream(
			func(
				ctx context.Context,
				args struct {
					Id string `json:"id"`
				},
			) (io.Reader, error) {
				return srv.Download(ctx, args.Id)
			},
		),
	)

	r.Register(
		"RpcTestService.Upload",
		processUploadAndReturn2(
			func(
				ctx context.Context,
				args struct {
					Id string
				},
				files func() (string, io.Reader, error),
			) ([]*File, error) {
				return srv.Upload(ctx, args.Id, files)
			},
		),
	)
}

func main() {
	server := httptest.NewServer(
		NewHttpReceiver(
			ReceiverFunc(
				func(ctx context.Context, in io.Reader, out io.Writer) {
					w, ok := out.(http.ResponseWriter)
					if !ok {
						return
					}

					pusher, err := sse.NewHttpPusher(w, 500*time.Millisecond)
					if err != nil {
						return
					}

					for range 100 {
						err = pusher.Push(sse.NewMessage("1", "hello", "world"))
						if err != nil {
							return
						}

						time.Sleep(2 * time.Second)
					}
				},
			),
		),
	)
	defer server.Close()

	fmt.Println(server.URL)

	select {}

	// dialer := NewHttpDialer(server.URL)

	// resp := dialer.Dial(context.Background(), strings.NewReader("hello"))
	// io.Copy(os.Stdout, resp)
}

//
// Helper functions
//

// SERVER

func caller1[A any](fn func(context.Context, A) error) Receiver {
	return ReceiverFunc(func(ctx context.Context, in io.Reader, out io.Writer) {
		var a A
		if err := json.NewDecoder(in).Decode(&a); err != nil {
			marshalError(out, err)
			return
		}

		marshalArray(out)(fn(ctx, a))
	})
}

func processAndReturn2[A, R1 any](fn func(context.Context, A) (R1, error)) Receiver {
	return ReceiverFunc(func(ctx context.Context, in io.Reader, out io.Writer) {
		var a A
		if err := json.NewDecoder(in).Decode(&a); err != nil {
			marshalError(out, err)
			return
		}

		marshalArray(out)(fn(ctx, a))
	})
}

func processStream[A any](fn func(context.Context, A) (<-chan *Event, error)) Receiver {
	return ReceiverFunc(func(ctx context.Context, in io.Reader, out io.Writer) {
		var a A
		if err := json.NewDecoder(in).Decode(&a); err != nil {
			marshalError(out, err)
			return
		}

		ch, err := fn(ctx, a)
		if err != nil {
			marshalError(out, err)
			return
		}

		pusher := sse.NewPushWriter(out, 500*time.Millisecond)

		for e := range ch {
			if err := pusher.Push(sse.NewMessage(e.Id, "data", e)); err != nil {
				return
			}
		}
	})
}

func processBinaryStream[A any](fn func(context.Context, A) (io.Reader, error)) Receiver {
	return ReceiverFunc(func(ctx context.Context, in io.Reader, out io.Writer) {
		var a A
		if err := json.NewDecoder(in).Decode(&a); err != nil {
			marshalError(out, err)
			return
		}

		r, err := fn(ctx, a)
		if err != nil {
			marshalError(out, err)
			return
		}

		if _, err := io.Copy(out, r); err != nil {
			marshalError(out, err)
			return
		}
	})
}

func processUploadAndReturn2[A any](fn func(context.Context, A, func() (string, io.Reader, error)) ([]*File, error)) Receiver {
	return ReceiverFunc(func(ctx context.Context, _ io.Reader, out io.Writer) {
		meta := MetaDataFromContext(ctx)
		if meta == nil {
			marshalError(out, errors.New("metadata not found"))
			return
		}

		idx := 0
		files := func() (string, io.Reader, error) {
			if idx >= len(meta.Files) {
				return "", nil, io.EOF
			}

			part := meta.Files[idx]
			idx++

			return part.FileName(), part, nil
		}

		var a A
		if err := json.Unmarshal(meta.Params, &a); err != nil {
			marshalError(out, err)
			return
		}

		marshalArray(out)(fn(ctx, a, files))
	})
}

func marshalError(out io.Writer, err error) error {
	return json.NewEncoder(out).Encode(err)
}

// last element in the array is error, if it's nil, it's a success
func marshalArray(out io.Writer) func(args ...any) {
	return func(args ...any) {
		if len(args) > 0 && args[len(args)-1] != nil {
			marshalError(out, args[len(args)-1].(error))
			return
		}

		json.NewEncoder(out).Encode(args)
	}
}
