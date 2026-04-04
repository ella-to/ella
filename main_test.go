package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ella.to/ella/compiler"
)

func TestGenCmd_GeneratesRuntimeTSForConsts(t *testing.T) {
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.ella")
	outDTS := filepath.Join(tmpDir, "schema.gen.d.ts")
	outTS := filepath.Join(tmpDir, "schema.gen.ts")

	source := `const TopicUserCreated = "jetdrive.user.created"`
	if err := os.WriteFile(schemaPath, []byte(source), 0o644); err != nil {
		t.Fatalf("failed writing schema: %v", err)
	}

	genCmd([]string{schemaPath}, "schema", outDTS, false, false)

	if _, err := os.Stat(outDTS); err != nil {
		t.Fatalf("expected declaration output file to exist: %v", err)
	}
	if _, err := os.Stat(outTS); err != nil {
		t.Fatalf("expected runtime output file to exist: %v", err)
	}
}

func TestGenCmd_SkipsRuntimeTSWhenNoConsts(t *testing.T) {
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.ella")
	outDTS := filepath.Join(tmpDir, "schema.gen.d.ts")
	outTS := filepath.Join(tmpDir, "schema.gen.ts")

	source := `model User {
	Id: string
}`
	if err := os.WriteFile(schemaPath, []byte(source), 0o644); err != nil {
		t.Fatalf("failed writing schema: %v", err)
	}

	genCmd([]string{schemaPath}, "schema", outDTS, false, false)

	if _, err := os.Stat(outDTS); err != nil {
		t.Fatalf("expected declaration output file to exist: %v", err)
	}
	if _, err := os.Stat(outTS); !os.IsNotExist(err) {
		t.Fatalf("expected runtime output file to be absent, got err=%v", err)
	}
}

func TestGenCmd_GeneratesRuntimeTSForErrors(t *testing.T) {
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.ella")
	outDTS := filepath.Join(tmpDir, "schema.gen.d.ts")
	outTS := filepath.Join(tmpDir, "schema.gen.ts")

	source := `error ErrNotFound { Msg = "resource not found" }`
	if err := os.WriteFile(schemaPath, []byte(source), 0o644); err != nil {
		t.Fatalf("failed writing schema: %v", err)
	}

	genCmd([]string{schemaPath}, "schema", outDTS, false, false)

	if _, err := os.Stat(outDTS); err != nil {
		t.Fatalf("expected declaration output file to exist: %v", err)
	}
	if _, err := os.Stat(outTS); err != nil {
		t.Fatalf("expected runtime output file to exist: %v", err)
	}
}

func TestGenCmd_GeneratesTypeScriptClientForTSOutput(t *testing.T) {
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.ella")
	outTS := filepath.Join(tmpDir, "schema.gen.ts")

	source := `model User {
	Id: string
}

service UserService {
	GetById (id: string) => (user: User)
}

error ErrUserNotFound { Msg = "user not found" }
`
	if err := os.WriteFile(schemaPath, []byte(source), 0o644); err != nil {
		t.Fatalf("failed writing schema: %v", err)
	}

	genCmd([]string{schemaPath}, "schema", outTS, false, false)

	b, err := os.ReadFile(outTS)
	if err != nil {
		t.Fatalf("expected ts output file to exist: %v", err)
	}

	code := string(b)
	if !strings.Contains(code, `export function createFetchJsonRpc(host: string, options: FetchJsonRpcOptions = {}): EllaRpcConnection {`) {
		t.Fatalf("expected fetch helper in ts client output, got:\n%s", code)
	}
	if !strings.Contains(code, `export function createUserService(conn: EllaRpcConnection): UserService {`) {
		t.Fatalf("expected createUserService factory in ts client output, got:\n%s", code)
	}
}

func TestHasConstDeclarations(t *testing.T) {
	progWithConst := parseProgramFromSource(t, `const Topic = "x"`)
	if !hasConstDeclarations(progWithConst) {
		t.Fatal("expected hasConstDeclarations to return true")
	}

	progWithoutConst := parseProgramFromSource(t, `model User {
	Id: string
}`)
	if hasConstDeclarations(progWithoutConst) {
		t.Fatal("expected hasConstDeclarations to return false")
	}
}

func TestHasErrorDeclarations(t *testing.T) {
	progWithError := parseProgramFromSource(t, `error ErrNotFound { Msg = "x" }`)
	if !hasErrorDeclarations(progWithError) {
		t.Fatal("expected hasErrorDeclarations to return true")
	}

	progWithoutError := parseProgramFromSource(t, `const Topic = "x"`)
	if hasErrorDeclarations(progWithoutError) {
		t.Fatal("expected hasErrorDeclarations to return false")
	}
}

func TestHasRuntimeTypeScriptExports(t *testing.T) {
	progWithConst := parseProgramFromSource(t, `const Topic = "x"`)
	if !hasRuntimeTypeScriptExports(progWithConst) {
		t.Fatal("expected hasRuntimeTypeScriptExports to return true for const")
	}

	progWithError := parseProgramFromSource(t, `error ErrNotFound { Msg = "x" }`)
	if !hasRuntimeTypeScriptExports(progWithError) {
		t.Fatal("expected hasRuntimeTypeScriptExports to return true for error")
	}

	progWithoutBoth := parseProgramFromSource(t, `model User {
	Id: string
}`)
	if hasRuntimeTypeScriptExports(progWithoutBoth) {
		t.Fatal("expected hasRuntimeTypeScriptExports to return false")
	}
}

func parseProgramFromSource(t *testing.T, source string) *compiler.Program {
	t.Helper()

	scanner := compiler.NewScanner(strings.NewReader(source), "test.ella")
	parser := compiler.NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	return program
}
