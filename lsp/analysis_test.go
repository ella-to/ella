package lsp

import (
	"strings"
	"testing"
)

func mkDoc(uri, text string) *Document {
	d := &Document{URI: uri, Path: URIToPath(uri), Text: withTestModule(text)}
	d.parse()
	return d
}

// withTestModule prepends the now-required `module test` declaration to a test
// source, unless it already declares a module. When the source begins with a
// newline the module replaces that blank first line, preserving content line
// numbers so position-based assertions keep working.
func withTestModule(text string) string {
	if strings.HasPrefix(strings.TrimLeft(text, " \t\r\n"), "module ") {
		return text
	}
	if strings.HasPrefix(text, "\n") {
		return "module test" + text
	}
	return "module test\n" + text
}

func indexOf(docs ...*Document) (*Index, map[string]*Document) {
	m := make(map[string]*Document, len(docs))
	for _, d := range docs {
		m[d.URI] = d
	}
	return buildIndex(m), m
}

// posOf returns the LSP position of the first occurrence of substr in text.
// Assumes ASCII (true for ella identifiers), so byte columns equal UTF-16 ones.
func posOf(t *testing.T, text, substr string) Position {
	t.Helper()
	i := strings.Index(text, substr)
	if i < 0 {
		t.Fatalf("substring %q not found in source", substr)
	}
	before := text[:i]
	line := strings.Count(before, "\n")
	col := len(before) - (strings.LastIndex(before, "\n") + 1)
	return Position{Line: line, Character: col}
}

func TestBuildIndexValidSchema(t *testing.T) {
	src := `
model User {
	Id: string
	Name: string
}

service UserService {
	GetById (id: string) => (user: User)
}
`
	idx, _ := indexOf(mkDoc("file:///a.ella", src))

	if got := idx.diagnostics["file:///a.ella"]; len(got) != 0 {
		t.Fatalf("expected no diagnostics, got %+v", got)
	}
	if idx.lookup("test", "User") == nil {
		t.Fatalf("expected User symbol in index")
	}
	if idx.lookup("test", "UserService") == nil {
		t.Fatalf("expected UserService symbol in index")
	}
	if idx.lookup("test", "User").Kind != kindModel {
		t.Fatalf("expected User to be a model, got %s", idx.lookup("test", "User").Kind)
	}
}

func TestBuildIndexUnknownTypeDiagnostic(t *testing.T) {
	src := `
model Account {
	Owner: Person
}
`
	idx, _ := indexOf(mkDoc("file:///a.ella", src))

	diags := idx.diagnostics["file:///a.ella"]
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d: %+v", len(diags), diags)
	}
	if !strings.Contains(diags[0].Message, "unknown type 'Person'") {
		t.Fatalf("unexpected diagnostic message: %q", diags[0].Message)
	}
	// The diagnostic should point at the "Person" token, not column 0.
	if diags[0].Range.Start.Character == 0 {
		t.Fatalf("expected a non-zero column for the Person reference, got %+v", diags[0].Range)
	}
}

func TestBuildIndexParseErrorDiagnostic(t *testing.T) {
	src := "model {\n}\n" // missing model name
	idx, _ := indexOf(mkDoc("file:///bad.ella", src))

	diags := idx.diagnostics["file:///bad.ella"]
	if len(diags) != 1 {
		t.Fatalf("expected 1 parse diagnostic, got %d: %+v", len(diags), diags)
	}
	if diags[0].Severity != severityError {
		t.Fatalf("expected error severity, got %d", diags[0].Severity)
	}
}

func TestCrossFileResolutionAndReferences(t *testing.T) {
	a := mkDoc("file:///models.ella", `
model User {
	Id: string
}
`)
	b := mkDoc("file:///services.ella", `
service UserService {
	GetById (id: string) => (user: User)
	List () => (users: []User)
}
`)
	idx, docs := indexOf(a, b)

	// No cross-file "unknown type" false positives.
	for uri, diags := range idx.diagnostics {
		if len(diags) != 0 {
			t.Fatalf("expected no diagnostics for %s, got %+v", uri, diags)
		}
	}

	s := &Server{docs: docs, index: idx}

	// Go-to-definition from the "User" usage in services.ella resolves to the
	// model declaration in models.ella.
	usagePos := posOf(t, b.Text, "User)")
	sym := s.resolveSymbol(b.URI, usagePos)
	if sym == nil {
		t.Fatalf("expected to resolve User usage to a symbol")
	}
	if sym.URI != a.URI {
		t.Fatalf("expected definition in %s, got %s", a.URI, sym.URI)
	}
	if sym.Name != "User" || sym.Kind != kindModel {
		t.Fatalf("expected model User, got %s %s", sym.Kind, sym.Name)
	}

	// find-references should report both usages of User.
	name := s.targetName(b.URI, usagePos)
	if name != "User" {
		t.Fatalf("expected target name User, got %q", name)
	}
	count := 0
	for _, u := range idx.usages {
		if u.Name == "User" {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("expected 2 usages of User, got %d", count)
	}
}

func TestModulesIsolateNames(t *testing.T) {
	// Two files in different modules reuse the name "User". This must not
	// produce duplicate-declaration diagnostics, and a "User" usage must resolve
	// within its own module only.
	a := mkDoc("file:///a.ella", "module a\nmodel User {\n\tId: string\n}\nservice S {\n\tGet () => (u: User)\n}\n")
	b := mkDoc("file:///b.ella", "module b\nmodel User {\n\tName: string\n}\n")
	idx, docs := indexOf(a, b)

	for uri, diags := range idx.diagnostics {
		if len(diags) != 0 {
			t.Fatalf("expected no diagnostics for %s, got %+v", uri, diags)
		}
	}

	if got := idx.lookup("a", "User"); got == nil || got.URI != a.URI {
		t.Fatalf("expected User in module a to resolve to a.ella, got %+v", got)
	}
	if got := idx.lookup("b", "User"); got == nil || got.URI != b.URI {
		t.Fatalf("expected User in module b to resolve to b.ella, got %+v", got)
	}

	// Go-to-definition from the usage in module a resolves to a's User, not b's.
	s := &Server{docs: docs, index: idx}
	pos := posOf(t, a.Text, "User)")
	sym := s.resolveSymbol(a.URI, pos)
	if sym == nil || sym.URI != a.URI {
		t.Fatalf("expected User usage in a.ella to resolve to a.ella, got %+v", sym)
	}
}

func TestModelExtendsUsage(t *testing.T) {
	src := `
model Base {
	Id: string
}

model Device {
	...Base
	Serial: string
}
`
	idx, docs := indexOf(mkDoc("file:///a.ella", src))
	s := &Server{docs: docs, index: idx}

	pos := posOf(t, src, "Base\n\tSerial") // the ...Base reference
	// Move into the Base identifier.
	pos.Character += 1
	sym := s.resolveSymbol("file:///a.ella", pos)
	if sym == nil || sym.Name != "Base" {
		t.Fatalf("expected ...Base to resolve to Base model, got %+v", sym)
	}
}

func TestConstReferenceUsage(t *testing.T) {
	src := `
const A = "x"
const B = A
`
	idx, docs := indexOf(mkDoc("file:///a.ella", src))
	s := &Server{docs: docs, index: idx}

	pos := posOf(t, src, "A\n") // the A used as B's value
	sym := s.resolveSymbol("file:///a.ella", pos)
	if sym == nil || sym.Name != "A" || sym.Kind != kindConst {
		t.Fatalf("expected reference to const A, got %+v", sym)
	}
}

func TestWordAt(t *testing.T) {
	text := "model User {\n\tId: string\n}"
	word, rng := wordAt(text, Position{Line: 0, Character: 8})
	if word != "User" {
		t.Fatalf("expected User, got %q", word)
	}
	if rng.Start.Character != 6 || rng.End.Character != 10 {
		t.Fatalf("unexpected range: %+v", rng)
	}

	// A position on a non-identifier character with no adjacent word yields
	// nothing (character 11 is the '{', preceded by a space).
	if w, _ := wordAt(text, Position{Line: 0, Character: 11}); w != "" {
		t.Fatalf("expected empty word on '{', got %q", w)
	}
}

func TestClassifyCompletion(t *testing.T) {
	if classifyCompletion("\tId: ") != ctxType {
		t.Fatalf("expected type context after colon")
	}
	if classifyCompletion("\tItems: []") != ctxType {
		t.Fatalf("expected type context inside array")
	}
	if classifyCompletion("model ") != ctxAny {
		t.Fatalf("expected any context at declaration start")
	}
}

func TestCompletionItems(t *testing.T) {
	idx, docs := indexOf(mkDoc("file:///a.ella", "model User {\n\tId: string\n}\nenum Status {\n\tA\n}\n"))
	s := &Server{docs: docs, index: idx}

	typeItems := s.completionItems(ctxType, "test")
	if !hasLabel(typeItems, "User") || !hasLabel(typeItems, "Status") || !hasLabel(typeItems, "string") {
		t.Fatalf("type completion missing expected items: %+v", labels(typeItems))
	}
	if hasLabel(typeItems, "model") {
		t.Fatalf("type completion should not offer the 'model' keyword")
	}

	anyItems := s.completionItems(ctxAny, "test")
	if !hasLabel(anyItems, "model") || !hasLabel(anyItems, "User") {
		t.Fatalf("any completion missing expected items: %+v", labels(anyItems))
	}
}

func TestDocumentSymbolOutline(t *testing.T) {
	src := `
enum Status {
	Active
	Disabled
}

model User {
	Id: string
	Name: string
}
`
	idx, _ := indexOf(mkDoc("file:///a.ella", src))
	syms := idx.symbolsForURI("file:///a.ella")
	if len(syms) != 2 {
		t.Fatalf("expected 2 top-level symbols, got %d", len(syms))
	}

	var enumSym, modelSym *Symbol
	for _, s := range syms {
		switch s.Name {
		case "Status":
			enumSym = s
		case "User":
			modelSym = s
		}
	}
	if enumSym == nil || modelSym == nil {
		t.Fatalf("missing expected symbols")
	}

	ds := documentSymbol(enumSym)
	if len(ds.Children) != 2 {
		t.Fatalf("expected 2 enum members, got %d", len(ds.Children))
	}

	dm := documentSymbol(modelSym)
	if len(dm.Children) != 2 {
		t.Fatalf("expected 2 model fields, got %d", len(dm.Children))
	}
	if dm.Children[0].Detail != "string" {
		t.Fatalf("expected field detail 'string', got %q", dm.Children[0].Detail)
	}
}

func hasLabel(items []CompletionItem, label string) bool {
	for _, it := range items {
		if it.Label == label {
			return true
		}
	}
	return false
}

func labels(items []CompletionItem) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Label)
	}
	return out
}
