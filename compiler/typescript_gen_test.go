package compiler

import (
	"strings"
	"testing"
)

func parseProgramForTypeScriptTest(t *testing.T, source string) *Program {
	t.Helper()

	scanner := NewScanner(strings.NewReader(withModule(source)), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	return program
}

func TestTypeScriptGenerator_ConstDeclarations(t *testing.T) {
	source := `const TopicUserCreated = "jetdrive.user.created"
const IsEnabled = true
`

	program := parseProgramForTypeScriptTest(t, source)
	gen := NewTypeScriptGenerator(program)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	if !strings.Contains(code, `export declare const TopicUserCreated: "jetdrive.user.created";`) {
		t.Fatalf("expected string const declaration in output, got:\n%s", code)
	}
	if !strings.Contains(code, `export declare const IsEnabled: true;`) {
		t.Fatalf("expected bool const declaration in output, got:\n%s", code)
	}
}

func TestTypeScriptGenerator_RuntimeConsts(t *testing.T) {
	source := `const TopicUserCreated = "jetdrive.user.created"
const MaxRetries = 3
const IsEnabled = true
const Nothing = null
`

	program := parseProgramForTypeScriptTest(t, source)
	gen := NewTypeScriptGenerator(program)

	var sb strings.Builder
	if err := gen.GenerateRuntimeConstsToWriter(&sb); err != nil {
		t.Fatalf("runtime const generation error: %v", err)
	}

	code := sb.String()
	if !strings.Contains(code, `export const TopicUserCreated = "jetdrive.user.created";`) {
		t.Fatalf("expected runtime string const export in output, got:\n%s", code)
	}
	if !strings.Contains(code, `export const MaxRetries = 3;`) {
		t.Fatalf("expected runtime number const export in output, got:\n%s", code)
	}
	if !strings.Contains(code, `export const IsEnabled = true;`) {
		t.Fatalf("expected runtime bool const export in output, got:\n%s", code)
	}
	if !strings.Contains(code, `export const Nothing = null;`) {
		t.Fatalf("expected runtime null const export in output, got:\n%s", code)
	}
}

func TestTypeScriptGenerator_RuntimeTemplateConst(t *testing.T) {
	source := `const TopicUserActionUpdated = "user.{{userId}}.{{action}}.updated"`

	program := parseProgramForTypeScriptTest(t, source)
	gen := NewTypeScriptGenerator(program)

	var sb strings.Builder
	if err := gen.GenerateRuntimeConstsToWriter(&sb); err != nil {
		t.Fatalf("runtime const generation error: %v", err)
	}

	code := sb.String()
	if !strings.Contains(code, `export function TopicUserActionUpdated(userId: string, action: string): string {`) {
		t.Fatalf("expected runtime template function signature in output, got:\n%s", code)
	}
	if !strings.Contains(code, "return `user.${userId}.${action}.updated`;") {
		t.Fatalf("expected runtime template function return in output, got:\n%s", code)
	}
}

func TestTypeScriptGenerator_RuntimeErrors(t *testing.T) {
	source := `error ErrNotFound { Msg = "resource not found" }
error ErrInvalidInput { Code = 400 Msg = "invalid input" }
`

	program := parseProgramForTypeScriptTest(t, source)
	gen := NewTypeScriptGenerator(program)

	var sb strings.Builder
	if err := gen.GenerateRuntimeConstsToWriter(&sb); err != nil {
		t.Fatalf("runtime const generation error: %v", err)
	}

	code := sb.String()
	if !strings.Contains(code, `export const ErrNotFound = 1000;`) {
		t.Fatalf("expected auto-assigned runtime error export in output, got:\n%s", code)
	}
	if !strings.Contains(code, `export const ErrInvalidInput = 400;`) {
		t.Fatalf("expected explicit runtime error export in output, got:\n%s", code)
	}
}

func TestTypeScriptGenerator_ClientRuntime(t *testing.T) {
	source := `const TopicUserAction = "user.{{userId}}.action"

enum UserStatus {
	Active = "active"
	Disabled = "disabled"
}

model User {
	Id: string
	Status: UserStatus
}

service UserService {
	GetById (id: string) => (user: User)
	Delete (id: string)
}

error ErrUserNotFound { Code = 404 Msg = "user not found" }
`

	program := parseProgramForTypeScriptTest(t, source)
	gen := NewTypeScriptGenerator(program)

	var sb strings.Builder
	if err := gen.GenerateClientToWriter(&sb); err != nil {
		t.Fatalf("client generation error: %v", err)
	}

	code := sb.String()
	if !strings.Contains(code, `export function createFetchJsonRpc(host: string, options: FetchJsonRpcOptions = {}): EllaRpcConnection {`) {
		t.Fatalf("expected fetch json-rpc helper in output, got:\n%s", code)
	}
	if !strings.Contains(code, `export function createUserService(conn: EllaRpcConnection): UserService {`) {
		t.Fatalf("expected service factory in output, got:\n%s", code)
	}
	if !strings.Contains(code, `"UserService.GetById"`) {
		t.Fatalf("expected json-rpc method wiring in output, got:\n%s", code)
	}
	if !strings.Contains(code, `export const ErrUserNotFound = 404;`) {
		t.Fatalf("expected runtime error constant in output, got:\n%s", code)
	}
	if !strings.Contains(code, `export function isErrUserNotFound(err: unknown): err is EllaRPCError {`) {
		t.Fatalf("expected per-error type guard in output, got:\n%s", code)
	}
}

func TestTypeScriptGenerator_ServiceOptionalArgs(t *testing.T) {
	source := `service UserService {
	List (query: string, limit?: int64, tags?: []string) => (ids: []string)
}

service GroupService {
	List (limit?: int64) => (ids: []string)
}
`

	program := parseProgramForTypeScriptTest(t, source)

	// Declaration output (.d.ts)
	gen := NewTypeScriptGenerator(program)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	if !strings.Contains(code, "export interface UserServiceListOptionalArgs {\n  limit?: number;\n  tags?: string[];\n}") {
		t.Fatalf("expected optional args interface in output, got:\n%s", code)
	}
	if !strings.Contains(code, "list(query: string, optionalArgs?: UserServiceListOptionalArgs, options?: OptionArgs): Promise<string[]>;") {
		t.Fatalf("expected method signature with optionalArgs in output, got:\n%s", code)
	}
	if !strings.Contains(code, "list(optionalArgs?: GroupServiceListOptionalArgs, options?: OptionArgs): Promise<string[]>;") {
		t.Fatalf("expected optional-only method signature in output, got:\n%s", code)
	}

	// Client output (.ts)
	var sb strings.Builder
	if err := gen.GenerateClientToWriter(&sb); err != nil {
		t.Fatalf("client generation error: %v", err)
	}

	clientCode := sb.String()
	if !strings.Contains(clientCode, "async list(query: string, optionalArgs?: UserServiceListOptionalArgs, options?: OptionArgs): Promise<string[]>") {
		t.Fatalf("expected factory method signature with optionalArgs in output, got:\n%s", clientCode)
	}
	if !strings.Contains(clientCode, "...optionalArgs,") {
		t.Fatalf("expected optionalArgs spread into params in output, got:\n%s", clientCode)
	}
}

func TestTypeScriptGenerator_ModelOptionalFields(t *testing.T) {
	source := `model Address {
	Street: string
}

model User {
	Name?: string
	Address?: Address
	Tags?: []string
}
`

	program := parseProgramForTypeScriptTest(t, source)
	gen := NewTypeScriptGenerator(program)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	if !strings.Contains(code, "name?: string;") {
		t.Fatalf("expected optional string field in output, got:\n%s", code)
	}
	if !strings.Contains(code, "address?: Address;") {
		t.Fatalf("expected optional model field in output, got:\n%s", code)
	}
	if !strings.Contains(code, "tags?: string[];") {
		t.Fatalf("expected optional array field in output, got:\n%s", code)
	}
}
