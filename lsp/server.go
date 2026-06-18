package lsp

import (
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"ella.to/ella/compiler"
)

func readFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// version is reported to the editor in the initialize result. It is overridden
// by the caller (main) so it tracks the CLI version.
var version = "dev"

// Server implements the ella language server. It is driven by a single
// goroutine that reads messages from the connection, handles them synchronously,
// and writes responses/notifications back. Because there is no concurrency the
// document store needs no locking.
type Server struct {
	conn   *Conn
	logger *log.Logger

	roots []string             // workspace root paths
	docs  map[string]*Document // by URI
	index *Index

	initialized bool
	shutdown    bool
	exit        bool
	exitCode    int
}

// Run starts the language server, reading from in and writing framed messages to
// out. Transport and diagnostic logging is written to logw (never to out, which
// carries the protocol). It returns when the client disconnects or sends exit.
func Run(in io.Reader, out io.Writer, logw io.Writer, ver string) error {
	if ver != "" {
		version = ver
	}
	logger := log.New(logw, "ella-lsp ", log.LstdFlags|log.Lmsgprefix)
	s := &Server{
		conn:   NewConn(in, out, logger),
		logger: logger,
		docs:   make(map[string]*Document),
		index:  &Index{symbolIndex: map[string]map[string]*Symbol{}, moduleByURI: map[string]string{}, diagnostics: map[string][]Diagnostic{}},
	}

	for {
		msg, err := s.conn.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			logger.Printf("read error: %v", err)
			return err
		}

		s.handle(msg)

		if s.exit {
			if s.exitCode != 0 {
				logger.Printf("exiting with code %d", s.exitCode)
			}
			return nil
		}
	}
}

func (s *Server) handle(msg *Message) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Printf("recovered from panic handling %q: %v", msg.Method, r)
			if msg.IsRequest() {
				s.conn.ReplyError(msg.ID, errInternalError, "internal server error")
			}
		}
	}()

	switch msg.Method {
	case "initialize":
		s.onInitialize(msg)
	case "initialized":
		s.onInitialized()
	case "shutdown":
		s.shutdown = true
		s.conn.Reply(msg.ID, nil)
	case "exit":
		s.exit = true
		if !s.shutdown {
			s.exitCode = 1
		}
	case "textDocument/didOpen":
		s.onDidOpen(msg)
	case "textDocument/didChange":
		s.onDidChange(msg)
	case "textDocument/didClose":
		s.onDidClose(msg)
	case "textDocument/didSave":
		// Full-sync already keeps text current; nothing extra to do.
	case "textDocument/definition":
		s.onDefinition(msg)
	case "textDocument/hover":
		s.onHover(msg)
	case "textDocument/documentSymbol":
		s.onDocumentSymbol(msg)
	case "textDocument/completion":
		s.onCompletion(msg)
	case "textDocument/references":
		s.onReferences(msg)
	case "textDocument/formatting":
		s.onFormatting(msg)
	default:
		// Unknown requests must be answered so the client does not hang;
		// unknown notifications are ignored.
		if msg.IsRequest() {
			s.conn.ReplyError(msg.ID, errMethodNotFound, "unsupported method: "+msg.Method)
		}
	}
}

func (s *Server) onInitialize(msg *Message) {
	var params InitializeParams
	_ = json.Unmarshal(msg.Params, &params)

	s.roots = collectRoots(params)
	s.logger.Printf("initialize: roots=%v", s.roots)

	result := InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync:           syncFull,
			DefinitionProvider:         true,
			HoverProvider:              true,
			DocumentSymbolProvider:     true,
			ReferencesProvider:         true,
			DocumentFormattingProvider: true,
			CompletionProvider:         &CompletionOptions{TriggerCharacters: []string{":", " ", "."}},
		},
		ServerInfo: ServerInfo{Name: "ella-lsp", Version: version},
	}
	s.conn.Reply(msg.ID, result)
}

func (s *Server) onInitialized() {
	s.initialized = true
	s.scanWorkspace()
	s.analyzeAndPublish()
}

func (s *Server) onDidOpen(msg *Message) {
	var params DidOpenTextDocumentParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return
	}
	td := params.TextDocument
	s.putDoc(td.URI, td.Text, true, td.Version)
	s.analyzeAndPublish()
}

func (s *Server) onDidChange(msg *Message) {
	var params DidChangeTextDocumentParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return
	}
	if len(params.ContentChanges) == 0 {
		return
	}
	// Full sync: the last change holds the entire document text.
	text := params.ContentChanges[len(params.ContentChanges)-1].Text
	s.putDoc(params.TextDocument.URI, text, true, params.TextDocument.Version)
	s.analyzeAndPublish()
}

func (s *Server) onDidClose(msg *Message) {
	var params DidCloseTextDocumentParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return
	}
	uri := params.TextDocument.URI
	// Keep the on-disk version available for cross-file analysis if the file
	// is inside a workspace root; otherwise drop it entirely.
	if d, ok := s.docs[uri]; ok {
		if content, err := readFile(d.Path); err == nil && s.withinRoots(d.Path) {
			d.Open = false
			d.Text = content
			d.parse()
		} else {
			// No longer tracked: clear its diagnostics so the editor does not
			// keep showing stale errors for a file we have dropped.
			delete(s.docs, uri)
			s.conn.Notify("textDocument/publishDiagnostics", PublishDiagnosticsParams{
				URI:         uri,
				Diagnostics: []Diagnostic{},
			})
		}
	}
	s.analyzeAndPublish()
}

func (s *Server) onDefinition(msg *Message) {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		s.conn.Reply(msg.ID, nil)
		return
	}
	sym := s.resolveSymbol(params.TextDocument.URI, params.Position)
	if sym == nil {
		s.conn.Reply(msg.ID, nil)
		return
	}
	s.conn.Reply(msg.ID, Location{URI: sym.URI, Range: sym.SelectionRange})
}

func (s *Server) onHover(msg *Message) {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		s.conn.Reply(msg.ID, nil)
		return
	}

	if sym := s.resolveSymbol(params.TextDocument.URI, params.Position); sym != nil {
		s.conn.Reply(msg.ID, Hover{
			Contents: MarkupContent{Kind: "markdown", Value: sym.hoverMarkdown()},
		})
		return
	}

	// Fall back to documentation for built-in types and keywords.
	if d := s.docs[params.TextDocument.URI]; d != nil {
		if word, rng := wordAt(d.Text, params.Position); word != "" {
			if doc, ok := builtinDocs[word]; ok {
				r := rng
				s.conn.Reply(msg.ID, Hover{
					Contents: MarkupContent{Kind: "markdown", Value: "**" + word + "**\n\n" + doc},
					Range:    &r,
				})
				return
			}
		}
	}

	s.conn.Reply(msg.ID, nil)
}

func (s *Server) onDocumentSymbol(msg *Message) {
	var params DocumentSymbolParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		s.conn.Reply(msg.ID, []DocumentSymbol{})
		return
	}
	symbols := s.index.symbolsForURI(params.TextDocument.URI)
	out := make([]DocumentSymbol, 0, len(symbols))
	for _, sym := range symbols {
		out = append(out, documentSymbol(sym))
	}
	s.conn.Reply(msg.ID, out)
}

func (s *Server) onCompletion(msg *Message) {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		s.conn.Reply(msg.ID, []CompletionItem{})
		return
	}
	d := s.docs[params.TextDocument.URI]
	if d == nil {
		s.conn.Reply(msg.ID, []CompletionItem{})
		return
	}
	ctx := classifyCompletion(linePrefix(d.Text, params.Position))
	s.conn.Reply(msg.ID, s.completionItems(ctx, s.index.moduleForURI(params.TextDocument.URI)))
}

func (s *Server) onReferences(msg *Message) {
	var params ReferenceParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		s.conn.Reply(msg.ID, []Location{})
		return
	}

	uri := params.TextDocument.URI
	name := s.targetName(uri, params.Position)
	if name == "" {
		s.conn.Reply(msg.ID, []Location{})
		return
	}

	// References resolve within a single module's namespace.
	module := s.index.moduleForURI(uri)

	locations := make([]Location, 0)
	for _, u := range s.index.usages {
		if u.Name == name && u.Module == module {
			locations = append(locations, Location{URI: u.URI, Range: u.Range})
		}
	}
	if params.Context.IncludeDeclaration {
		if sym := s.index.lookup(module, name); sym != nil {
			locations = append(locations, Location{URI: sym.URI, Range: sym.SelectionRange})
		}
	}
	s.conn.Reply(msg.ID, locations)
}

func (s *Server) onFormatting(msg *Message) {
	var params DocumentFormattingParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		s.conn.Reply(msg.ID, []TextEdit{})
		return
	}
	d := s.docs[params.TextDocument.URI]
	if d == nil || d.parseErr != nil || d.prog == nil {
		// Cannot format a document that does not parse.
		s.conn.Reply(msg.ID, []TextEdit{})
		return
	}
	formatted := compiler.Format(d.prog)
	s.conn.Reply(msg.ID, []TextEdit{{
		Range:   wholeDocumentRange(d.Text),
		NewText: formatted,
	}})
}

//
// Resolution helpers shared by definition / hover / references
//

// resolveSymbol finds the declaration referenced at pos: first an explicit
// usage, then the declaration name itself, then a best-effort match on the word
// under the cursor (which keeps things working when the file does not fully
// parse).
func (s *Server) resolveSymbol(uri string, pos Position) *Symbol {
	module := s.index.moduleForURI(uri)
	if u := s.index.usageAt(uri, pos); u != nil {
		if sym := s.index.lookup(module, u.Name); sym != nil {
			return sym
		}
	}
	if sym := s.index.symbolAt(uri, pos); sym != nil {
		return sym
	}
	if d := s.docs[uri]; d != nil {
		if word, _ := wordAt(d.Text, pos); word != "" {
			if sym := s.index.lookup(module, word); sym != nil {
				return sym
			}
		}
	}
	return nil
}

// targetName returns the symbol name the cursor is on, for find-references.
func (s *Server) targetName(uri string, pos Position) string {
	if u := s.index.usageAt(uri, pos); u != nil {
		return u.Name
	}
	if sym := s.index.symbolAt(uri, pos); sym != nil {
		return sym.Name
	}
	if d := s.docs[uri]; d != nil {
		if word, _ := wordAt(d.Text, pos); word != "" {
			if s.index.lookup(s.index.moduleForURI(uri), word) != nil {
				return word
			}
		}
	}
	return ""
}

//
// Document store / workspace
//

func (s *Server) putDoc(uri, text string, open bool, version int) {
	d := &Document{
		URI:     uri,
		Path:    URIToPath(uri),
		Text:    text,
		Version: version,
		Open:    open,
	}
	d.parse()
	s.docs[uri] = d
}

func (s *Server) analyzeAndPublish() {
	s.index = buildIndex(s.docs)
	for uri := range s.docs {
		diags := s.index.diagnostics[uri]
		if diags == nil {
			diags = []Diagnostic{}
		}
		s.conn.Notify("textDocument/publishDiagnostics", PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: diags,
		})
	}
}

// scanWorkspace discovers every .ella file under the workspace roots and loads
// it (unless already open) so cross-file analysis works before each file is
// opened in the editor.
func (s *Server) scanWorkspace() {
	for _, root := range s.roots {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if shouldSkipDir(d.Name()) && path != root {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".ella") {
				return nil
			}
			uri := PathToURI(path)
			if _, ok := s.docs[uri]; ok {
				return nil // already tracked (likely open)
			}
			content, readErr := readFile(path)
			if readErr != nil {
				s.logger.Printf("failed to read %s: %v", path, readErr)
				return nil
			}
			s.putDoc(uri, content, false, 0)
			return nil
		})
	}
}

func (s *Server) withinRoots(path string) bool {
	for _, root := range s.roots {
		if rel, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(rel, "..") {
			return true
		}
	}
	return false
}

//
// Completion + document symbol builders
//

var builtinTypeNames = []string{
	"string", "bool", "byte", "timestamp", "any",
	"int8", "int16", "int32", "int64",
	"uint8", "uint16", "uint32", "uint64",
	"float32", "float64", "map",
}

var topLevelKeywords = []string{"const", "enum", "model", "service", "error"}

func (s *Server) completionItems(ctx completionContext, module string) []CompletionItem {
	items := make([]CompletionItem, 0)

	if ctx == ctxAny {
		for _, kw := range topLevelKeywords {
			items = append(items, CompletionItem{Label: kw, Kind: completionKindKeyword, Detail: "keyword"})
		}
	}

	for _, t := range builtinTypeNames {
		items = append(items, CompletionItem{Label: t, Kind: completionKindTypeParam, Detail: "built-in type"})
	}

	// Named types (and, outside type context, consts) declared in the same
	// module, since references never cross module boundaries.
	for _, sym := range s.index.symbols {
		if sym.Module != module {
			continue
		}
		switch sym.Kind {
		case kindModel:
			items = append(items, CompletionItem{Label: sym.Name, Kind: completionKindStruct, Detail: "model"})
		case kindEnum:
			items = append(items, CompletionItem{Label: sym.Name, Kind: completionKindEnum, Detail: "enum"})
		case kindConst:
			if ctx == ctxAny {
				items = append(items, CompletionItem{Label: sym.Name, Kind: completionKindConstant, Detail: "const"})
			}
		}
	}

	return items
}

func documentSymbol(sym *Symbol) DocumentSymbol {
	ds := DocumentSymbol{
		Name:           sym.Name,
		Kind:           lspSymbolKind(sym.Kind),
		Range:          sym.Range,
		SelectionRange: sym.SelectionRange,
	}

	switch n := sym.node.(type) {
	case *compiler.ConstDecl:
		ds.Detail = n.Assignment.Value.String()
	case *compiler.DeclEnum:
		for _, val := range n.Values {
			child := DocumentSymbol{
				Name:           val.Name.Name,
				Kind:           symbolKindEnumMember,
				Range:          tokenRange(val.Name.Token),
				SelectionRange: tokenRange(val.Name.Token),
			}
			if val.IsDefined && val.Value != nil {
				child.Detail = val.Value.String()
			}
			ds.Children = append(ds.Children, child)
		}
	case *compiler.DeclModel:
		ds.Detail = "model"
		for _, field := range n.Fields {
			ds.Children = append(ds.Children, DocumentSymbol{
				Name:           field.Name.Name,
				Detail:         field.Type.String(),
				Kind:           symbolKindField,
				Range:          tokenRange(field.Name.Token),
				SelectionRange: tokenRange(field.Name.Token),
			})
		}
	case *compiler.DeclService:
		ds.Detail = "service"
		for _, method := range n.Methods {
			ds.Children = append(ds.Children, DocumentSymbol{
				Name:           method.Name.Name,
				Detail:         methodSignature(method),
				Kind:           symbolKindMethod,
				Range:          tokenRange(method.Name.Token),
				SelectionRange: tokenRange(method.Name.Token),
			})
		}
	case *compiler.DeclError:
		ds.Detail = errorDetail(n)
	}

	return ds
}

func lspSymbolKind(kind symbolKind) int {
	switch kind {
	case kindConst:
		return symbolKindConstant
	case kindEnum:
		return symbolKindEnum
	case kindModel:
		return symbolKindStruct
	case kindService:
		return symbolKindInterface
	case kindError:
		return symbolKindObject
	}
	return symbolKindObject
}

func methodSignature(m *compiler.DeclServiceMethod) string {
	var sb strings.Builder
	sb.WriteString("(")
	for i, arg := range m.Args {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(arg.String())
	}
	sb.WriteString(")")
	if len(m.Returns) > 0 {
		sb.WriteString(" => (")
		for i, ret := range m.Returns {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(ret.String())
		}
		sb.WriteString(")")
	}
	return sb.String()
}

func errorDetail(e *compiler.DeclError) string {
	var parts []string
	if e.Code != nil {
		parts = append(parts, "Code = "+e.Code.String())
	}
	if e.Msg != nil {
		parts = append(parts, "Msg = "+e.Msg.String())
	}
	return strings.Join(parts, " ")
}

//
// URI <-> path
//

// PathToURI converts a filesystem path to a file:// URI.
func PathToURI(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	u := url.URL{Scheme: "file", Path: path}
	return u.String()
}

// URIToPath converts a file:// URI to a filesystem path. Non-file URIs are
// returned unchanged.
func URIToPath(uri string) string {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" {
		return uri
	}
	return u.Path
}

func collectRoots(params InitializeParams) []string {
	var roots []string
	seen := make(map[string]bool)
	add := func(p string) {
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		roots = append(roots, p)
	}

	for _, wf := range params.WorkspaceFolders {
		add(URIToPath(wf.URI))
	}
	if params.RootURI != "" {
		add(URIToPath(params.RootURI))
	}
	add(params.RootPath)
	return roots
}

func shouldSkipDir(name string) bool {
	switch name {
	case "node_modules", "vendor", ".git":
		return true
	}
	return strings.HasPrefix(name, ".") && name != "."
}

func wholeDocumentRange(text string) Range {
	lines := strings.Split(text, "\n")
	lastLine := len(lines) - 1
	last := lines[lastLine]
	last = strings.TrimSuffix(last, "\r")
	return Range{
		Start: Position{Line: 0, Character: 0},
		End:   Position{Line: lastLine, Character: utf16Len(last)},
	}
}
