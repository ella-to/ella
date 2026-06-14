package lsp

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"
)

func TestConnRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	w := NewConn(nil, &buf, nil)
	w.Notify("test/method", map[string]string{"hello": "world"})

	r := NewConn(&buf, io.Discard, nil)
	msg, err := r.Read()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if msg.Method != "test/method" {
		t.Fatalf("expected method test/method, got %q", msg.Method)
	}
	var params map[string]string
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	if params["hello"] != "world" {
		t.Fatalf("unexpected params: %+v", params)
	}
}

func TestReplyNullResult(t *testing.T) {
	var buf bytes.Buffer
	w := NewConn(nil, &buf, nil)
	w.Reply(json.RawMessage("1"), nil)

	// The body must contain "result":null (not omit it) so JSON-RPC clients
	// treat shutdown / empty-definition replies as successful.
	if !bytes.Contains(buf.Bytes(), []byte(`"result":null`)) {
		t.Fatalf("expected result:null in body, got: %s", buf.String())
	}
}

// rpcClient drives the server over in-memory pipes for end-to-end tests.
type rpcClient struct {
	t    *testing.T
	conn *Conn
}

func (c *rpcClient) request(id int, method string, params any) {
	if err := c.conn.write(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}); err != nil {
		c.t.Fatalf("write request %s: %v", method, err)
	}
}

func (c *rpcClient) notify(method string, params any) {
	if err := c.conn.write(map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}); err != nil {
		c.t.Fatalf("write notification %s: %v", method, err)
	}
}

// read returns the next message, failing the test if none arrives quickly.
func (c *rpcClient) read() *Message {
	c.t.Helper()
	type result struct {
		msg *Message
		err error
	}
	ch := make(chan result, 1)
	go func() {
		msg, err := c.conn.Read()
		ch <- result{msg, err}
	}()
	select {
	case r := <-ch:
		if r.err != nil {
			c.t.Fatalf("read: %v", r.err)
		}
		return r.msg
	case <-time.After(3 * time.Second):
		c.t.Fatalf("timed out waiting for a message")
		return nil
	}
}

func TestServerEndToEnd(t *testing.T) {
	clientReader, serverWriter := io.Pipe() // server -> client
	serverReader, clientWriter := io.Pipe() // client -> server

	done := make(chan error, 1)
	go func() {
		done <- Run(serverReader, serverWriter, io.Discard, "test")
	}()

	client := &rpcClient{t: t, conn: NewConn(clientReader, clientWriter, nil)}

	// initialize
	client.request(1, "initialize", InitializeParams{})
	resp := client.read()
	var initResult InitializeResult
	if err := json.Unmarshal(resp.Result, &initResult); err != nil {
		t.Fatalf("unmarshal initialize result: %v", err)
	}
	if !initResult.Capabilities.DefinitionProvider {
		t.Fatalf("expected definitionProvider capability")
	}
	if initResult.Capabilities.TextDocumentSync != syncFull {
		t.Fatalf("expected full text sync")
	}

	// initialized (no roots -> no diagnostics published)
	client.notify("initialized", map[string]any{})

	// open a document with a model used by a service in the same file
	src := "model User {\n\tId: string\n}\nservice S {\n\tGet (id: string) => (user: User)\n}\n"
	client.notify("textDocument/didOpen", DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{URI: "file:///x.ella", LanguageID: "ella", Version: 1, Text: src},
	})

	// expect a publishDiagnostics notification (empty, since the schema is valid)
	diagMsg := client.read()
	if diagMsg.Method != "textDocument/publishDiagnostics" {
		t.Fatalf("expected publishDiagnostics, got %q", diagMsg.Method)
	}
	var diagParams PublishDiagnosticsParams
	if err := json.Unmarshal(diagMsg.Params, &diagParams); err != nil {
		t.Fatalf("unmarshal diagnostics: %v", err)
	}
	if len(diagParams.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %+v", diagParams.Diagnostics)
	}

	// go-to-definition on the "User" return type (line 4, inside "User")
	client.request(2, "textDocument/definition", TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///x.ella"},
		Position:     Position{Line: 4, Character: 30},
	})
	defResp := client.read()
	var loc Location
	if err := json.Unmarshal(defResp.Result, &loc); err != nil {
		t.Fatalf("unmarshal definition result: %v (raw: %s)", err, defResp.Result)
	}
	if loc.URI != "file:///x.ella" || loc.Range.Start.Line != 0 {
		t.Fatalf("expected definition at model User (line 0), got %+v", loc)
	}

	// document symbols
	client.request(3, "textDocument/documentSymbol", DocumentSymbolParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///x.ella"},
	})
	symResp := client.read()
	var syms []DocumentSymbol
	if err := json.Unmarshal(symResp.Result, &syms); err != nil {
		t.Fatalf("unmarshal symbols: %v", err)
	}
	if len(syms) != 2 {
		t.Fatalf("expected 2 document symbols, got %d: %+v", len(syms), syms)
	}

	// shutdown + exit
	client.request(4, "shutdown", nil)
	shutResp := client.read()
	if shutResp.Error != nil {
		t.Fatalf("shutdown returned error: %+v", shutResp.Error)
	}
	client.notify("exit", nil)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("server exited with error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("server did not exit after exit notification")
	}
}
