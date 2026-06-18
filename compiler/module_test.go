package compiler

import (
	"strings"
	"testing"
)

func parse(t *testing.T, src, name string) *Program {
	t.Helper()
	prog, err := NewParser(NewScanner(strings.NewReader(src), name)).Parse()
	if err != nil {
		t.Fatalf("unexpected parse error for %s: %v", name, err)
	}
	return prog
}

// merge concatenates programs the way the CLI (gen) and the LSP do before
// validation: per-file module declarations stay in Nodes as boundary markers.
func merge(progs ...*Program) *Program {
	var merged Program
	for _, p := range progs {
		merged.Nodes = append(merged.Nodes, p.Nodes...)
		merged.Comments = append(merged.Comments, p.Comments...)
	}
	return &merged
}

func TestModule_RequiredFirstDeclaration(t *testing.T) {
	_, err := NewParser(NewScanner(strings.NewReader("const A = 1\n"), "test.ella")).Parse()
	if err == nil {
		t.Fatal("expected error when a file does not start with a module declaration")
	}
	if !strings.Contains(err.Error(), "module") {
		t.Fatalf("expected a module-related error, got: %v", err)
	}
}

func TestModule_ParsesAsFirstNode(t *testing.T) {
	prog := parse(t, "module foo\nconst A = 1\n", "test.ella")
	if prog.Module != "foo" {
		t.Fatalf("expected program module 'foo', got %q", prog.Module)
	}
	if len(prog.Nodes) != 2 {
		t.Fatalf("expected 2 nodes (module + const), got %d", len(prog.Nodes))
	}
	mod, ok := prog.Nodes[0].(*DeclModule)
	if !ok {
		t.Fatalf("expected first node to be *DeclModule, got %T", prog.Nodes[0])
	}
	if mod.Name.Name != "foo" {
		t.Fatalf("expected module name 'foo', got %q", mod.Name.Name)
	}
}

func TestModule_CommentsMayPrecede(t *testing.T) {
	prog := parse(t, "# leading comment\nmodule foo\nconst A = 1\n", "test.ella")
	if prog.Module != "foo" {
		t.Fatalf("expected module 'foo', got %q", prog.Module)
	}
}

func TestModule_DeclarationCannotFollowOther(t *testing.T) {
	_, err := NewParser(NewScanner(strings.NewReader("module foo\nconst A = 1\nmodule bar\n"), "test.ella")).Parse()
	if err == nil {
		t.Fatal("expected error for a second module declaration")
	}
	if !strings.Contains(err.Error(), "first declaration") {
		t.Fatalf("expected a 'first declaration' error, got: %v", err)
	}
}

func TestModule_EmptyFileAllowed(t *testing.T) {
	prog := parse(t, "\n  \n", "test.ella")
	if prog.Module != "" {
		t.Fatalf("expected empty module for an empty file, got %q", prog.Module)
	}
	if len(prog.Nodes) != 0 {
		t.Fatalf("expected no nodes for an empty file, got %d", len(prog.Nodes))
	}
}

func TestModule_SameNamesInDifferentModulesDoNotConflict(t *testing.T) {
	a := parse(t, "module a\nmodel User { Id: string }\n", "a.ella")
	b := parse(t, "module b\nmodel User { Name: string }\n", "b.ella")

	errs := ValidateProgram(merge(a, b))
	if len(errs) != 0 {
		t.Fatalf("expected no errors for same names across modules, got: %v", errs)
	}
}

func TestModule_SameModuleAcrossFilesStillConflicts(t *testing.T) {
	a := parse(t, "module shared\nmodel User { Id: string }\n", "a.ella")
	b := parse(t, "module shared\nmodel User { Name: string }\n", "b.ella")

	errs := ValidateProgram(merge(a, b))
	if len(errs) == 0 {
		t.Fatal("expected a duplicate model error within the same module")
	}
	if !strings.Contains(errs[0].Error(), "duplicate model") {
		t.Fatalf("expected duplicate model error, got: %v", errs[0])
	}
}

func TestModule_CrossModuleTypeReferenceIsUnknown(t *testing.T) {
	// Address is defined in module b; module a may not reference it.
	a := parse(t, "module a\nmodel User { Addr: Address }\n", "a.ella")
	b := parse(t, "module b\nmodel Address { Street: string }\n", "b.ella")

	errs := ValidateProgram(merge(a, b))
	if len(errs) == 0 {
		t.Fatal("expected an unknown type error for a cross-module reference")
	}
	if !strings.Contains(errs[0].Error(), "unknown type 'Address'") {
		t.Fatalf("expected unknown type error, got: %v", errs[0])
	}
}

func TestModule_SameModuleTypeReferenceResolves(t *testing.T) {
	a := parse(t, "module app\nmodel User { Addr: Address }\n", "a.ella")
	b := parse(t, "module app\nmodel Address { Street: string }\n", "b.ella")

	errs := ValidateProgram(merge(a, b))
	if len(errs) != 0 {
		t.Fatalf("expected no errors when type is defined in the same module, got: %v", errs)
	}
}

func TestModule_FormatPlacesModuleFirst(t *testing.T) {
	prog := parse(t, "module foo\nconst A = 1\n", "test.ella")
	got := Format(prog)
	want := "module foo\n\nconst A = 1"
	if got != want {
		t.Fatalf("unexpected format output:\n got: %q\nwant: %q", got, want)
	}
}
