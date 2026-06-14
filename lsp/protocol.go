package lsp

// This file defines the subset of the Language Server Protocol types that the
// ella language server speaks. Only the fields we actually read or write are
// included; everything else in an incoming message is ignored by the JSON
// decoder. See https://microsoft.github.io/language-server-protocol/ for the
// full specification.

// Position is a zero-based line and character offset. Per the LSP spec the
// character offset is measured in UTF-16 code units.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range is a half-open span between two positions.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location points at a range inside a specific document.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// --- Lifecycle ---

type WorkspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

type InitializeParams struct {
	ProcessID        *int              `json:"processId"`
	RootURI          string            `json:"rootUri"`
	RootPath         string            `json:"rootPath"`
	WorkspaceFolders []WorkspaceFolder `json:"workspaceFolders"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   ServerInfo         `json:"serverInfo"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// TextDocumentSyncKind values. We use Full so every change resends the whole
// document, which keeps the server simple and robust.
const (
	syncNone        = 0
	syncFull        = 1
	syncIncremental = 2
)

type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

type ServerCapabilities struct {
	TextDocumentSync           int                `json:"textDocumentSync"`
	DefinitionProvider         bool               `json:"definitionProvider"`
	HoverProvider              bool               `json:"hoverProvider"`
	DocumentSymbolProvider     bool               `json:"documentSymbolProvider"`
	ReferencesProvider         bool               `json:"referencesProvider"`
	DocumentFormattingProvider bool               `json:"documentFormattingProvider"`
	CompletionProvider         *CompletionOptions `json:"completionProvider,omitempty"`
}

// --- Text synchronization ---

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// TextDocumentContentChangeEvent carries a change. With full sync the Range is
// absent and Text is the entire new document.
type TextDocumentContentChangeEvent struct {
	Range *Range `json:"range,omitempty"`
	Text  string `json:"text"`
}

type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         *string                `json:"text,omitempty"`
}

// --- Diagnostics ---

// DiagnosticSeverity values.
const (
	severityError       = 1
	severityWarning     = 2
	severityInformation = 3
	severityHint        = 4
)

type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity,omitempty"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message"`
}

type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// --- Hover ---

type MarkupContent struct {
	Kind  string `json:"kind"` // "plaintext" or "markdown"
	Value string `json:"value"`
}

type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

// --- Document symbols ---

// SymbolKind values used by the outline.
const (
	symbolKindMethod     = 6
	symbolKindField      = 8
	symbolKindEnum       = 10
	symbolKindInterface  = 11
	symbolKindConstant   = 14
	symbolKindStruct     = 23
	symbolKindEnumMember = 22
	symbolKindObject     = 19
)

type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           int              `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// --- Completion ---

// CompletionItemKind values.
const (
	completionKindMethod    = 2
	completionKindField     = 5
	completionKindClass     = 7
	completionKindEnum      = 13
	completionKindKeyword   = 14
	completionKindConstant  = 21
	completionKindStruct    = 22
	completionKindTypeParam = 25
)

type CompletionItem struct {
	Label         string `json:"label"`
	Kind          int    `json:"kind,omitempty"`
	Detail        string `json:"detail,omitempty"`
	Documentation string `json:"documentation,omitempty"`
	InsertText    string `json:"insertText,omitempty"`
}

// --- References ---

type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

type ReferenceParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      ReferenceContext       `json:"context"`
}

// --- Formatting ---

type DocumentFormattingParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}
