package lsp

import (
	"fmt"
	"strings"

	"ella.to/ella/compiler"
)

// Document is a single tracked .ella file. Every known file in the workspace
// (whether opened by the editor or discovered on disk) has a Document. The
// parsed program and parse error are cached so a single keystroke only re-parses
// the file that changed.
type Document struct {
	URI      string
	Path     string
	Text     string
	Version  int
	Open     bool // true when the editor has the file open
	prog     *compiler.Program
	parseErr error
}

// parse re-parses the document text, caching the resulting program (or error).
func (d *Document) parse() {
	d.prog, d.parseErr = parseSource(d.Path, d.Text)
}

func parseSource(path, text string) (*compiler.Program, error) {
	return compiler.NewParser(compiler.NewScanner(strings.NewReader(text), path)).Parse()
}

// symbolKind enumerates the top-level declaration kinds ella supports.
type symbolKind string

const (
	kindConst   symbolKind = "const"
	kindEnum    symbolKind = "enum"
	kindModel   symbolKind = "model"
	kindService symbolKind = "service"
	kindError   symbolKind = "error"
)

// Symbol is a top-level declaration that other declarations can refer to by
// name (a const, enum, model, service, or error). Module records the namespace
// it was declared in so lookups stay within a single module.
type Symbol struct {
	Name           string
	Module         string
	Kind           symbolKind
	URI            string
	Range          Range // the whole declaration
	SelectionRange Range // just the name
	node           compiler.Node
}

// Reference is a place in the source that names a top-level Symbol: a custom
// type used in a field/argument/return, a model "...Extend", or a const used as
// an option/const value. Module is the namespace of the file the reference
// appears in.
type Reference struct {
	Name   string
	Module string
	URI    string
	Range  Range
}

// Index is the analysis result for the whole workspace. It is rebuilt whenever
// any document changes.
type Index struct {
	symbols     []*Symbol
	symbolIndex map[string]map[string]*Symbol // module -> name -> symbol
	usages      []*Reference
	moduleByURI map[string]string       // document URI -> module name
	diagnostics map[string][]Diagnostic // keyed by document URI
}

// lookup returns the symbol with the given name declared in the given module,
// or nil. Resolution never crosses module boundaries.
func (idx *Index) lookup(module, name string) *Symbol {
	if byName, ok := idx.symbolIndex[module]; ok {
		return byName[name]
	}
	return nil
}

// moduleForURI returns the module a document belongs to (empty if unknown).
func (idx *Index) moduleForURI(uri string) string {
	return idx.moduleByURI[uri]
}

// buildIndex parses (from cache), merges and validates every document, then
// builds the cross-file symbol and reference tables and per-file diagnostics.
func buildIndex(docs map[string]*Document) *Index {
	idx := &Index{
		symbolIndex: make(map[string]map[string]*Symbol),
		moduleByURI: make(map[string]string),
		diagnostics: make(map[string][]Diagnostic),
	}

	// path -> uri so validation errors (which only carry the source path) can be
	// mapped back to the document that owns them.
	pathToURI := make(map[string]string, len(docs))
	for _, d := range docs {
		pathToURI[d.Path] = d.URI
	}
	resolveURI := func(src string) string {
		if uri, ok := pathToURI[src]; ok {
			return uri
		}
		return PathToURI(src)
	}

	var merged compiler.Program

	for uri, d := range docs {
		// Ensure every document gets a diagnostics entry so stale diagnostics are
		// cleared even when a file becomes clean.
		idx.diagnostics[uri] = nil

		if d.parseErr != nil {
			idx.addDiag(uri, diagnosticFromError(d.parseErr))
			continue
		}
		if d.prog != nil {
			merged.Nodes = append(merged.Nodes, d.prog.Nodes...)
			merged.Comments = append(merged.Comments, d.prog.Comments...)
		}
	}

	for _, err := range compiler.ValidateProgram(&merged) {
		ce, ok := err.(*compiler.Error)
		if !ok || ce.Token == nil {
			continue
		}
		idx.addDiag(resolveURI(ce.Token.Pos.Src), diagnosticFromError(ce))
	}

	currentModule := ""
	for _, node := range merged.Nodes {
		if mod, ok := node.(*compiler.DeclModule); ok {
			currentModule = mod.Name.Name
			if mod.Token != nil {
				idx.moduleByURI[resolveURI(mod.Token.Pos.Src)] = currentModule
			}
			continue
		}
		if sym := symbolFromNode(node, currentModule, resolveURI); sym != nil {
			idx.symbols = append(idx.symbols, sym)
			byName := idx.symbolIndex[currentModule]
			if byName == nil {
				byName = make(map[string]*Symbol)
				idx.symbolIndex[currentModule] = byName
			}
			if _, exists := byName[sym.Name]; !exists {
				byName[sym.Name] = sym
			}
		}
		idx.usages = append(idx.usages, usagesFromNode(node, currentModule, resolveURI)...)
	}

	return idx
}

func (idx *Index) addDiag(uri string, d Diagnostic) {
	idx.diagnostics[uri] = append(idx.diagnostics[uri], d)
}

// usageAt returns the reference whose range contains pos in the given document.
func (idx *Index) usageAt(uri string, pos Position) *Reference {
	for _, u := range idx.usages {
		if u.URI == uri && rangeContains(u.Range, pos) {
			return u
		}
	}
	return nil
}

// symbolAt returns the symbol whose name token is under pos in the given
// document (i.e. the cursor is on the declaration itself).
func (idx *Index) symbolAt(uri string, pos Position) *Symbol {
	for _, s := range idx.symbols {
		if s.URI == uri && rangeContains(s.SelectionRange, pos) {
			return s
		}
	}
	return nil
}

// symbolsForURI returns the declarations defined in a single document, used to
// build the document outline.
func (idx *Index) symbolsForURI(uri string) []*Symbol {
	var out []*Symbol
	for _, s := range idx.symbols {
		if s.URI == uri {
			out = append(out, s)
		}
	}
	return out
}

//
// Position / token helpers
//

func diagnosticFromError(err error) Diagnostic {
	ce, ok := err.(*compiler.Error)
	if !ok {
		return Diagnostic{Severity: severityError, Source: "ella", Message: err.Error()}
	}
	rng := Range{}
	if ce.Token != nil && !ce.Token.IsInjected() && ce.Token.Pos.Line >= 1 {
		rng = tokenRange(ce.Token)
	}
	return Diagnostic{
		Range:    rng,
		Severity: severityError,
		Source:   "ella",
		Message:  ce.Reason,
	}
}

// tokenRange converts a compiler token to an LSP range. It accounts for tokens
// whose literal spans multiple lines (e.g. backtick strings).
func tokenRange(tok *compiler.Token) Range {
	start := Position{Line: tok.Pos.Line - 1, Character: tok.Pos.Column - 1}
	if start.Line < 0 {
		start.Line = 0
	}
	if start.Character < 0 {
		start.Character = 0
	}

	lit := tok.Lit
	if lit == "" {
		return Range{Start: start, End: Position{Line: start.Line, Character: start.Character + 1}}
	}

	if nl := strings.Count(lit, "\n"); nl > 0 {
		lastLine := lit[strings.LastIndex(lit, "\n")+1:]
		return Range{Start: start, End: Position{Line: start.Line + nl, Character: utf16Len(lastLine)}}
	}

	return Range{Start: start, End: Position{Line: start.Line, Character: start.Character + utf16Len(lit)}}
}

func rangeContains(r Range, pos Position) bool {
	if pos.Line < r.Start.Line || pos.Line > r.End.Line {
		return false
	}
	if pos.Line == r.Start.Line && pos.Character < r.Start.Character {
		return false
	}
	if pos.Line == r.End.Line && pos.Character > r.End.Character {
		return false
	}
	return true
}

//
// Symbol / reference collection from the AST
//

func symbolFromNode(node compiler.Node, module string, resolveURI func(string) string) *Symbol {
	switch n := node.(type) {
	case *compiler.ConstDecl:
		nameTok := n.Assignment.Name.Token
		end := tokenRange(nameTok).End
		if vt := exprToken(n.Assignment.Value); vt != nil {
			end = tokenRange(vt).End
		}
		return &Symbol{
			Name:           n.Assignment.Name.Name,
			Module:         module,
			Kind:           kindConst,
			URI:            resolveURI(nameTok.Pos.Src),
			Range:          Range{Start: tokenRange(n.Token).Start, End: end},
			SelectionRange: tokenRange(nameTok),
			node:           n,
		}
	case *compiler.DeclEnum:
		return declSymbol(kindEnum, module, n.Token, n.Name.Token, n.CloseCurly, n, resolveURI)
	case *compiler.DeclModel:
		return declSymbol(kindModel, module, n.Token, n.Name.Token, n.CloseCurly, n, resolveURI)
	case *compiler.DeclService:
		return declSymbol(kindService, module, n.Token, n.Name.Token, n.CloseCurly, n, resolveURI)
	case *compiler.DeclError:
		return declSymbol(kindError, module, n.Token, n.Name.Token, n.CloseCurly, n, resolveURI)
	}
	return nil
}

// declSymbol builds a Symbol for a block declaration that runs from a keyword
// token to its closing brace.
func declSymbol(kind symbolKind, module string, keyword, name, closeCurly *compiler.Token, node compiler.Node, resolveURI func(string) string) *Symbol {
	rng := Range{Start: tokenRange(keyword).Start, End: tokenRange(name).End}
	if closeCurly != nil {
		rng.End = tokenRange(closeCurly).End
	}
	return &Symbol{
		Name:           name.Lit,
		Module:         module,
		Kind:           kind,
		URI:            resolveURI(name.Pos.Src),
		Range:          rng,
		SelectionRange: tokenRange(name),
		node:           node,
	}
}

func usagesFromNode(node compiler.Node, module string, resolveURI func(string) string) []*Reference {
	var refs []*Reference

	add := func(name string, tok *compiler.Token) {
		if tok == nil || tok.IsInjected() {
			return
		}
		refs = append(refs, &Reference{Name: name, URI: resolveURI(tok.Pos.Src), Range: tokenRange(tok)})
	}

	switch n := node.(type) {
	case *compiler.ConstDecl:
		// A const can reference another const by name as its value.
		if iden, ok := n.Assignment.Value.(*compiler.IdenExpr); ok {
			add(iden.Name, iden.Token)
		}
	case *compiler.DeclModel:
		for _, ext := range n.Extends {
			add(ext.Name, ext.Token)
		}
		for _, field := range n.Fields {
			refs = append(refs, typeUsages(field.Type, resolveURI)...)
			refs = append(refs, optionUsages(field.Options, resolveURI)...)
		}
	case *compiler.DeclService:
		for _, method := range n.Methods {
			for _, arg := range method.Args {
				refs = append(refs, typeUsages(arg.Type, resolveURI)...)
			}
			for _, ret := range method.Returns {
				refs = append(refs, typeUsages(ret.Type, resolveURI)...)
			}
			refs = append(refs, optionUsages(method.Options, resolveURI)...)
		}
	}

	// Stamp the enclosing module on every reference so find-references and
	// go-to-definition resolve within the right namespace.
	for _, r := range refs {
		r.Module = module
	}

	return refs
}

// typeUsages walks a type declaration and records references to named
// (custom) types, descending through arrays and maps.
func typeUsages(t compiler.Decl, resolveURI func(string) string) []*Reference {
	switch dt := t.(type) {
	case *compiler.DeclCustomType:
		if dt.Name.Token == nil || dt.Name.Token.IsInjected() {
			return nil
		}
		return []*Reference{{Name: dt.Name.Name, URI: resolveURI(dt.Name.Token.Pos.Src), Range: tokenRange(dt.Name.Token)}}
	case *compiler.DeclArrayType:
		return typeUsages(dt.Type, resolveURI)
	case *compiler.DeclMapType:
		return append(typeUsages(dt.KeyType, resolveURI), typeUsages(dt.ValueType, resolveURI)...)
	}
	return nil
}

func optionUsages(options []*compiler.AssignmentStmt, resolveURI func(string) string) []*Reference {
	var refs []*Reference
	for _, opt := range options {
		if iden, ok := opt.Value.(*compiler.IdenExpr); ok && iden.Token != nil && !iden.Token.IsInjected() {
			refs = append(refs, &Reference{Name: iden.Name, URI: resolveURI(iden.Token.Pos.Src), Range: tokenRange(iden.Token)})
		}
	}
	return refs
}

func exprToken(e compiler.Expr) *compiler.Token {
	switch v := e.(type) {
	case *compiler.IdenExpr:
		return v.Token
	case *compiler.ValueExprNumber:
		return v.Token
	case *compiler.ValueExprString:
		return v.Token
	case *compiler.ValueExprBool:
		return v.Token
	case *compiler.ValueExprNull:
		return v.Token
	}
	return nil
}

//
// Hover rendering
//

// hoverMarkdown renders a markdown hover body for a symbol: a one-line summary
// plus the full declaration as a fenced ella code block.
func (s *Symbol) hoverMarkdown() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "**%s** `%s`\n\n", s.Kind, s.Name)
	sb.WriteString("```ella\n")
	sb.WriteString(s.node.String())
	sb.WriteString("\n```")
	return sb.String()
}

// builtinDocs maps built-in type names and keywords to a short hover blurb.
var builtinDocs = map[string]string{
	"string":    "Built-in type: UTF-8 text.",
	"bool":      "Built-in type: boolean (`true` / `false`).",
	"byte":      "Built-in type: a single byte.",
	"timestamp": "Built-in type: a point in time (Unix timestamp).",
	"any":       "Built-in type: untyped value (maps to `interface{}` / `any`).",
	"int8":      "Built-in type: signed 8-bit integer.",
	"int16":     "Built-in type: signed 16-bit integer.",
	"int32":     "Built-in type: signed 32-bit integer.",
	"int64":     "Built-in type: signed 64-bit integer.",
	"uint8":     "Built-in type: unsigned 8-bit integer.",
	"uint16":    "Built-in type: unsigned 16-bit integer.",
	"uint32":    "Built-in type: unsigned 32-bit integer.",
	"uint64":    "Built-in type: unsigned 64-bit integer.",
	"float32":   "Built-in type: 32-bit floating point.",
	"float64":   "Built-in type: 64-bit floating point.",
	"map":       "Built-in type: `map<KeyType, ValueType>`.",
	"module":    "Keyword: declares the namespace a file belongs to. Must be the first declaration in the file; only comments may precede it.",
	"const":     "Keyword: declares a named constant value.",
	"enum":      "Keyword: declares an enumeration (integer or string valued).",
	"model":     "Keyword: declares a data structure with typed fields.",
	"service":   "Keyword: declares an RPC service with methods.",
	"error":     "Keyword: declares a named error with an optional code and message.",
}

//
// Word extraction and completion context
//

// wordAt returns the identifier word at pos and its range. An identifier is a
// run of letters, digits, and underscores.
func wordAt(text string, pos Position) (string, Range) {
	line := lineAt(text, pos.Line)
	byteIdx := utf16OffsetToByte(line, pos.Character)

	start := byteIdx
	for start > 0 && isIdentByte(line[start-1]) {
		start--
	}
	end := byteIdx
	for end < len(line) && isIdentByte(line[end]) {
		end++
	}
	if start == end {
		return "", Range{}
	}

	word := line[start:end]
	rng := Range{
		Start: Position{Line: pos.Line, Character: byteToUTF16(line, start)},
		End:   Position{Line: pos.Line, Character: byteToUTF16(line, end)},
	}
	return word, rng
}

// completionContext classifies the line prefix before the cursor so the server
// can offer the most relevant completions.
type completionContext int

const (
	ctxAny  completionContext = iota // top of a declaration: keywords + types + names
	ctxType                          // after a ':' or inside map<>/[] : types + named types
)

func classifyCompletion(linePrefix string) completionContext {
	if strings.ContainsAny(linePrefix, ":") || strings.Contains(linePrefix, "map<") || strings.Contains(linePrefix, "[]") {
		return ctxType
	}
	return ctxAny
}

// linePrefix returns the text of the given line up to (not including) the
// character offset.
func linePrefix(text string, pos Position) string {
	line := lineAt(text, pos.Line)
	byteIdx := utf16OffsetToByte(line, pos.Character)
	return line[:byteIdx]
}

func lineAt(text string, line int) string {
	lines := strings.Split(text, "\n")
	if line < 0 || line >= len(lines) {
		return ""
	}
	return strings.TrimSuffix(lines[line], "\r")
}

func isIdentByte(b byte) bool {
	return b == '_' ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9')
}

//
// UTF-16 conversions (LSP measures characters in UTF-16 code units)
//

func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		if r > 0xFFFF {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// utf16OffsetToByte maps a UTF-16 code-unit offset within s to a byte index.
func utf16OffsetToByte(s string, off int) int {
	if off <= 0 {
		return 0
	}
	u := 0
	for i, r := range s {
		if u >= off {
			return i
		}
		if r > 0xFFFF {
			u += 2
		} else {
			u++
		}
	}
	return len(s)
}

func byteToUTF16(s string, b int) int {
	if b > len(s) {
		b = len(s)
	}
	return utf16Len(s[:b])
}
