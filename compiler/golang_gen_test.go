package compiler

import (
	"strings"
	"testing"
)

func TestGoGenerator_Const(t *testing.T) {
	source := `const MaxSize = 100kb
const Timeout = 1m
const Topic = "test.topic"
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := NewGoGenerator(program, "main")
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	// Check const declarations
	if !strings.Contains(code, "const MaxSize = 102400") {
		t.Errorf("expected MaxSize = 102400 in output, got:\n%s", code)
	}
	if !strings.Contains(code, "time.Minute") {
		t.Errorf("expected time.Minute in output, got:\n%s", code)
	}
	if !strings.Contains(code, `const Topic = "test.topic"`) {
		t.Errorf("expected Topic const in output, got:\n%s", code)
	}

	t.Logf("Generated code:\n%s", code)
}

func TestGoGenerator_StringConsts(t *testing.T) {
	source := `const TopicUserCreated = "jetdrive.user.created"
const TopicUserStatusUpdated = "jetdrive.user.status.updated"
const TopicUserDeleted = "jetdrive.user.deleted"
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := NewGoGenerator(program, "main")
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	// Check string const declarations
	if !strings.Contains(code, `const TopicUserCreated = "jetdrive.user.created"`) {
		t.Errorf("expected TopicUserCreated const in output, got:\n%s", code)
	}
	if !strings.Contains(code, `const TopicUserStatusUpdated = "jetdrive.user.status.updated"`) {
		t.Errorf("expected TopicUserStatusUpdated const in output, got:\n%s", code)
	}
	if !strings.Contains(code, `const TopicUserDeleted = "jetdrive.user.deleted"`) {
		t.Errorf("expected TopicUserDeleted const in output, got:\n%s", code)
	}

	t.Logf("Generated code:\n%s", code)
}

func TestGoGenerator_TemplateStringConsts(t *testing.T) {
	source := `const TopicUserCreated = "user.{{userId}}.created"
const TopicUserActionUpdated = "user.{{userId}}.{{action}}.updated"
const TopicGlobal = "global.event"
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := NewGoGenerator(program, "main")
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	// Check template string becomes function with single param
	if !strings.Contains(code, `func TopicUserCreated(userId string) string {`) {
		t.Errorf("expected TopicUserCreated function in output, got:\n%s", code)
	}
	if !strings.Contains(code, `return "user." + userId + ".created"`) {
		t.Errorf("expected proper return statement for TopicUserCreated, got:\n%s", code)
	}

	// Check template string becomes function with multiple params
	if !strings.Contains(code, `func TopicUserActionUpdated(userId string, action string) string {`) {
		t.Errorf("expected TopicUserActionUpdated function in output, got:\n%s", code)
	}
	if !strings.Contains(code, `return "user." + userId + "." + action + ".updated"`) {
		t.Errorf("expected proper return statement for TopicUserActionUpdated, got:\n%s", code)
	}

	// Check non-template string remains as const
	if !strings.Contains(code, `const TopicGlobal = "global.event"`) {
		t.Errorf("expected TopicGlobal const in output, got:\n%s", code)
	}

	t.Logf("Generated code:\n%s", code)
}

func TestGoGenerator_Enum(t *testing.T) {
	source := `enum Status {
	Pending
	Active
	Completed
}

enum ErrorCode {
	NotFound = 404
	Internal = 500
}

enum StringEnum {
	Red = "red"
	Green = "green"
	Blue = "blue"
}
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := NewGoGenerator(program, "main")
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	// Check type declarations
	if !strings.Contains(code, "type Status int") {
		t.Errorf("expected Status type in output, got:\n%s", code)
	}
	if !strings.Contains(code, "type StringEnum string") {
		t.Errorf("expected StringEnum type in output, got:\n%s", code)
	}
	// Const names now have underscore: Status_Pending
	if !strings.Contains(code, "Status_Pending") {
		t.Errorf("expected Status_Pending const in output, got:\n%s", code)
	}
	// Check explicit values are generated with type on all values
	if !strings.Contains(code, "Status_Pending   Status = 0") {
		t.Errorf("expected Status_Pending with explicit value 0, got:\n%s", code)
	}
	if !strings.Contains(code, "Status_Active    Status = 1") {
		t.Errorf("expected Status_Active with type and value 1, got:\n%s", code)
	}
	// Check String() method
	if !strings.Contains(code, "func (s Status) String() string") {
		t.Errorf("expected Status String() method in output, got:\n%s", code)
	}
	// Check MarshalJSON method
	if !strings.Contains(code, "func (s Status) MarshalJSON() ([]byte, error)") {
		t.Errorf("expected Status MarshalJSON() method in output, got:\n%s", code)
	}
	// Check UnmarshalJSON method
	if !strings.Contains(code, "func (s *Status) UnmarshalJSON(data []byte) error") {
		t.Errorf("expected Status UnmarshalJSON() method in output, got:\n%s", code)
	}
	// Check that error is returned on unknown value
	if !strings.Contains(code, `unknown Status value`) {
		t.Errorf("expected unknown Status value error in output, got:\n%s", code)
	}

	t.Logf("Generated code:\n%s", code)
}

func TestGoGenerator_EnumWithPlaceholder(t *testing.T) {
	// Note: This test requires the parser to support '_' as a valid enum value name.
	// Currently the parser only accepts IDENTIFIER tokens for enum values.
	// Once parser is updated to support '_', this test should work.
	t.Skip("Parser does not support '_' as enum value yet")

	source := `enum Priority {
	_
	Low
	Medium
	High
	Critical
}
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := NewGoGenerator(program, "main")
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	// Check type declaration
	if !strings.Contains(code, "type Priority int") {
		t.Errorf("expected Priority type in output, got:\n%s", code)
	}
	// Check underscore placeholder (first value with iota)
	if !strings.Contains(code, "_ Priority = iota") {
		t.Errorf("expected _ Priority = iota in output, got:\n%s", code)
	}
	// Check named values use underscore separator
	if !strings.Contains(code, "Priority_Low") {
		t.Errorf("expected Priority_Low const in output, got:\n%s", code)
	}
	if !strings.Contains(code, "Priority_Critical") {
		t.Errorf("expected Priority_Critical const in output, got:\n%s", code)
	}
	// String() method should not include the placeholder
	if strings.Contains(code, `return "_"`) {
		t.Errorf("String() method should not return underscore, got:\n%s", code)
	}
	// Check String() returns value names
	if !strings.Contains(code, `return "Low"`) {
		t.Errorf("expected String() to return Low, got:\n%s", code)
	}
	// UnmarshalJSON should not include the placeholder case
	if strings.Contains(code, `case "_":`) {
		t.Errorf("UnmarshalJSON should not have case for underscore, got:\n%s", code)
	}

	t.Logf("Generated code:\n%s", code)
}

func TestGoGenerator_Model(t *testing.T) {
	source := `model User {
	Id: string
	Name: string
	Age: int32
	Data: byte
	CreatedAt: timestamp
	Tags: []string
	Metadata: map<string, string>
}
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := NewGoGenerator(program, "main")
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	// Check struct declaration
	if !strings.Contains(code, "type User struct") {
		t.Errorf("expected User struct in output, got:\n%s", code)
	}
	if !strings.Contains(code, `json:"id"`) {
		t.Errorf("expected json tag in output, got:\n%s", code)
	}
	if !strings.Contains(code, "Data") || !strings.Contains(code, "byte") {
		t.Errorf("expected Data byte in output, got:\n%s", code)
	}
	if !strings.Contains(code, "CreatedAt") || !strings.Contains(code, "time.Time") {
		t.Errorf("expected CreatedAt time.Time in output, got:\n%s", code)
	}
	if !strings.Contains(code, "[]string") {
		t.Errorf("expected []string in output, got:\n%s", code)
	}
	if !strings.Contains(code, "map[string]string") {
		t.Errorf("expected map[string]string in output, got:\n%s", code)
	}

	t.Logf("Generated code:\n%s", code)
}

func TestGoGenerator_ModelWithEnumAndModelRef(t *testing.T) {
	source := `enum Status {
	Active
	Inactive
}

model Address {
	Street: string
	City: string
}

model User {
	Id: string
	Status: Status
	Address: Address
}
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := NewGoGenerator(program, "main")
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	// Check that enum reference is not a pointer
	if !strings.Contains(code, "Status  Status") {
		t.Errorf("expected Status field with Status type in output, got:\n%s", code)
	}
	// Check that model reference is a pointer
	if !strings.Contains(code, "Address *Address") {
		t.Errorf("expected Address field with *Address type in output, got:\n%s", code)
	}
	// Check enum const names use underscore
	if !strings.Contains(code, "Status_Active") {
		t.Errorf("expected Status_Active const in output, got:\n%s", code)
	}

	t.Logf("Generated code:\n%s", code)
}

func TestGoGenerator_Service(t *testing.T) {
	source := `service GreetingService {
	SayHello(name: string) => (result: string)
	SayBye(name: string) => (result: string)
}
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := NewGoGenerator(program, "main")
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	// Check interface
	if !strings.Contains(code, "type GreetingService interface") {
		t.Errorf("expected GreetingService interface in output, got:\n%s", code)
	}
	if !strings.Contains(code, "SayHello(ctx context.Context, name string) (string, error)") {
		t.Errorf("expected SayHello method signature in output, got:\n%s", code)
	}

	// Check server
	if !strings.Contains(code, "greetingServiceServer") {
		t.Errorf("expected server struct in output, got:\n%s", code)
	}
	if !strings.Contains(code, "RegisterGreetingServiceServer") {
		t.Errorf("expected register function in output, got:\n%s", code)
	}

	// Check client
	if !strings.Contains(code, "greetingServiceClient") {
		t.Errorf("expected client struct in output, got:\n%s", code)
	}
	if !strings.Contains(code, "CreateGreetingServiceClient") {
		t.Errorf("expected create client function in output, got:\n%s", code)
	}

	t.Logf("Generated code:\n%s", code)
}

func TestGoGenerator_ServiceEnumReturnZeroValues(t *testing.T) {
	source := `enum RoleType {
	Any = -1
	Owner = 0
	Admin = 1
}

enum RoleLabel {
	Any = "any"
	Owner = "owner"
	Admin = "admin"
}

service RoleService {
	GetRoleType(userId: string) => (roleType: RoleType)
	GetRoleLabel(userId: string) => (roleLabel: RoleLabel)
}
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := NewGoGenerator(program, "main")
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	if !strings.Contains(code, "func (c *roleServiceClient) GetRoleType(ctx context.Context, userId string) (RoleType, error)") {
		t.Fatalf("expected GetRoleType client method in output, got:\n%s", code)
	}
	if !strings.Contains(code, "return 0, err") {
		t.Errorf("expected numeric enum zero value return for GetRoleType, got:\n%s", code)
	}

	if !strings.Contains(code, "func (c *roleServiceClient) GetRoleLabel(ctx context.Context, userId string) (RoleLabel, error)") {
		t.Fatalf("expected GetRoleLabel client method in output, got:\n%s", code)
	}
	if !strings.Contains(code, `return "", err`) {
		t.Errorf("expected string enum zero value return for GetRoleLabel, got:\n%s", code)
	}
}

func TestGoGenerator_Error(t *testing.T) {
	source := `error ErrNotFound { Msg = "resource not found" }
error ErrInvalidInput { Code = 400 Msg = "invalid input" }
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := NewGoGenerator(program, "main")
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	// Check error variables
	if !strings.Contains(code, "var ErrNotFound") {
		t.Errorf("expected ErrNotFound var in output, got:\n%s", code)
	}
	if !strings.Contains(code, "var ErrInvalidInput") {
		t.Errorf("expected ErrInvalidInput var in output, got:\n%s", code)
	}
	if !strings.Contains(code, "jsonrpc.NewError(400,") {
		t.Errorf("expected jsonrpc.NewError(400, ...) in output, got:\n%s", code)
	}
	if !strings.Contains(code, `"resource not found"`) {
		t.Errorf("expected error message in output, got:\n%s", code)
	}

	t.Logf("Generated code:\n%s", code)
}

func TestGoGenerator_CompleteExample(t *testing.T) {
	source := `const MaxLogoAssetSize = 100kb
const TimeoutLogoAsset = 1m
const TopicBusinessCreated = "rentify.business.created"

enum Status {
	Pending
	Active
	Completed
}

model Attribute {
	Key: string
	Value: string
}

model Business {
	Id: string
	Name: string
	Status: Status
	Attributes: []Attribute
	CreatedOn: timestamp
	UpdatedOn: timestamp
}

service BusinessService {
	Create (name: string, attributes: []Attribute) => (result: Business)
	Update (business: Business) => (result: Business)
	Delete (businessId: string)
	GetById (businessId: string) => (result: Business)
}

error ErrBusinessNameMissing { Msg = "business name is required" }
error ErrBusinessNotFound { Msg = "business not found" }
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := NewGoGenerator(program, "main")
	code, err := gen.GenerateWithHelpers()
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	// Basic checks
	if !strings.Contains(code, "package main") {
		t.Errorf("expected package declaration in output")
	}
	if !strings.Contains(code, "import") {
		t.Errorf("expected import declaration in output")
	}

	t.Logf("Generated code:\n%s", code)
}
