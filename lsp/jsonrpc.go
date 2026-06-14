package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
)

// Message is a single JSON-RPC 2.0 frame. A frame may be a request (has ID and
// Method), a notification (Method, no ID), or a response (ID plus Result or
// Error). The fields are decoded lazily so a handler can unmarshal Params into
// the concrete type it expects.
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ResponseError  `json:"error,omitempty"`
}

// IsRequest reports whether the message expects a response (it carries an id
// and a method).
func (m *Message) IsRequest() bool {
	return len(m.ID) > 0 && m.Method != ""
}

// IsNotification reports whether the message is a fire-and-forget notification.
func (m *Message) IsNotification() bool {
	return len(m.ID) == 0 && m.Method != ""
}

// ResponseError is the error object of a JSON-RPC response.
type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Standard JSON-RPC / LSP error codes.
const (
	errParseError     = -32700
	errInvalidRequest = -32600
	errMethodNotFound = -32601
	errInvalidParams  = -32602
	errInternalError  = -32603
)

// Conn is a JSON-RPC 2.0 connection over a reader/writer pair using the LSP
// base protocol framing (Content-Length headers followed by a JSON body).
type Conn struct {
	r      *bufio.Reader
	w      io.Writer
	wmu    sync.Mutex
	logger *log.Logger
}

// NewConn builds a connection from a reader (incoming messages) and a writer
// (outgoing messages). The logger receives transport-level diagnostics and must
// not write to w (typically it points at stderr).
func NewConn(r io.Reader, w io.Writer, logger *log.Logger) *Conn {
	return &Conn{
		r:      bufio.NewReader(r),
		w:      w,
		logger: logger,
	}
}

// Read blocks until a full message frame is available and decodes it. It
// returns io.EOF when the peer closes the stream.
func (c *Conn) Read() (*Message, error) {
	contentLength := -1

	for {
		line, err := c.r.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break // end of headers
		}

		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(name), "content-length") {
			contentLength, err = strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length header: %w", err)
			}
		}
	}

	if contentLength < 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(c.r, body); err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, fmt.Errorf("invalid JSON-RPC body: %w", err)
	}

	return &msg, nil
}

// write frames and sends a single value as one JSON-RPC message. It is safe to
// call from multiple goroutines.
func (c *Conn) write(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	c.wmu.Lock()
	defer c.wmu.Unlock()

	if _, err := fmt.Fprintf(c.w, "Content-Length: %d\r\n\r\n", len(data)); err != nil {
		return err
	}
	if _, err := c.w.Write(data); err != nil {
		return err
	}
	return nil
}

// successResponse and errorResponse are split so that a successful reply always
// emits "result" (even when null) while an error reply emits only "error", as
// required by JSON-RPC 2.0.
type successResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result"`
}

type errorResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Error   *ResponseError  `json:"error"`
}

type notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

// Reply sends a successful response for the request with the given id. A nil
// result is serialized as JSON null, which is what several LSP requests expect
// when there is nothing to return.
func (c *Conn) Reply(id json.RawMessage, result any) {
	if err := c.write(successResponse{JSONRPC: "2.0", ID: id, Result: result}); err != nil {
		c.logf("failed to write response: %v", err)
	}
}

// ReplyError sends an error response for the request with the given id.
func (c *Conn) ReplyError(id json.RawMessage, code int, message string) {
	if err := c.write(errorResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &ResponseError{Code: code, Message: message},
	}); err != nil {
		c.logf("failed to write error response: %v", err)
	}
}

// Notify sends a notification (no response expected) to the peer.
func (c *Conn) Notify(method string, params any) {
	if err := c.write(notification{JSONRPC: "2.0", Method: method, Params: params}); err != nil {
		c.logf("failed to write notification %q: %v", method, err)
	}
}

func (c *Conn) logf(format string, args ...any) {
	if c.logger != nil {
		c.logger.Printf(format, args...)
	}
}
