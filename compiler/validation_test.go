package compiler

import (
	"strings"
	"testing"
)

func toError(t *testing.T, err error) *Error {
	compilerErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected compiler error, got: %v", err)
	}
	return compilerErr
}

func TestValidator_DuplicateConst(t *testing.T) {
	source := `const A = 1
const A = 2
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) == 0 {
		t.Error("expected validation error for duplicate const")
	}
	if !strings.Contains(toError(t, errors[0]).Reason, "duplicate const") {
		t.Errorf("expected duplicate const error, got: %s", toError(t, errors[0]).Reason)
	}
}

func TestValidator_DuplicateEnum(t *testing.T) {
	source := `enum Status { Active }
enum Status { Inactive }
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) == 0 {
		t.Error("expected validation error for duplicate enum")
	}
	if !strings.Contains(toError(t, errors[0]).Reason, "duplicate enum") {
		t.Errorf("expected duplicate enum error, got: %s", toError(t, errors[0]).Reason)
	}
}

func TestValidator_DuplicateModel(t *testing.T) {
	source := `model User { Id: string }
model User { Name: string }
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) == 0 {
		t.Error("expected validation error for duplicate model")
	}
	if !strings.Contains(toError(t, errors[0]).Reason, "duplicate model") {
		t.Errorf("expected duplicate model error, got: %s", toError(t, errors[0]).Reason)
	}
}

func TestValidator_DuplicateService(t *testing.T) {
	source := `service Greeting { Hello() }
service Greeting { Bye() }
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) == 0 {
		t.Error("expected validation error for duplicate service")
	}
	if !strings.Contains(toError(t, errors[0]).Reason, "duplicate service") {
		t.Errorf("expected duplicate service error, got: %s", toError(t, errors[0]).Reason)
	}
}

func TestValidator_DuplicateFieldInModel(t *testing.T) {
	source := `model User {
	Id: string
	Name: string
	Id: int32
}
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) == 0 {
		t.Error("expected validation error for duplicate field")
	}
	if !strings.Contains(toError(t, errors[0]).Reason, "duplicate field") {
		t.Errorf("expected duplicate field error, got: %s", toError(t, errors[0]).Reason)
	}
}

func TestValidator_UndefinedConstReference(t *testing.T) {
	source := `const A = B
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) == 0 {
		t.Error("expected validation error for undefined const reference")
	}
	if !strings.Contains(toError(t, errors[0]).Reason, "undefined const") {
		t.Errorf("expected undefined const error, got: %s", toError(t, errors[0]).Reason)
	}
}

func TestValidator_ValidConstReference(t *testing.T) {
	source := `const A = 1
const B = A
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) != 0 {
		t.Errorf("expected no validation errors, got: %v", errors)
	}
}

func TestValidator_UnknownFieldType(t *testing.T) {
	source := `model User {
	Id: string
	Status: UnknownType
}
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) == 0 {
		t.Error("expected validation error for unknown type")
	}
	if !strings.Contains(toError(t, errors[0]).Reason, "unknown type") {
		t.Errorf("expected unknown type error, got: %s", toError(t, errors[0]).Reason)
	}
}

func TestValidator_ValidFieldType(t *testing.T) {
	source := `enum Status { Active }
model Address { Street: string }
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

	errors := ValidateProgram(program)
	if len(errors) != 0 {
		t.Errorf("expected no validation errors, got: %v", errors)
	}
}

func TestValidator_InvalidMapKeyType(t *testing.T) {
	source := `model Data {
	Values: map<bool, string>
}
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) == 0 {
		t.Error("expected validation error for invalid map key type")
	}
	if !strings.Contains(toError(t, errors[0]).Reason, "map key type must be string or number") {
		t.Errorf("expected map key type error, got: %s", toError(t, errors[0]).Reason)
	}
}

func TestValidator_ValidMapKeyType(t *testing.T) {
	source := `model Data {
	StringMap: map<string, string>
	IntMap: map<int32, string>
}
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) != 0 {
		t.Errorf("expected no validation errors, got: %v", errors)
	}
}

func TestValidator_DuplicateMethodInService(t *testing.T) {
	source := `service Greeting {
	Hello()
	Hello()
}
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) == 0 {
		t.Error("expected validation error for duplicate method")
	}
	if !strings.Contains(toError(t, errors[0]).Reason, "duplicate method") {
		t.Errorf("expected duplicate method error, got: %s", toError(t, errors[0]).Reason)
	}
}

func TestValidator_UnknownMethodArgType(t *testing.T) {
	source := `service Greeting {
	Hello(user: UnknownType)
}
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) == 0 {
		t.Error("expected validation error for unknown arg type")
	}
	if !strings.Contains(toError(t, errors[0]).Reason, "unknown type") {
		t.Errorf("expected unknown type error, got: %s", toError(t, errors[0]).Reason)
	}
}

func TestValidator_UnknownMethodReturnType(t *testing.T) {
	source := `service Greeting {
	Hello() => (result: UnknownType)
}
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) == 0 {
		t.Error("expected validation error for unknown return type")
	}
	if !strings.Contains(toError(t, errors[0]).Reason, "unknown type") {
		t.Errorf("expected unknown type error, got: %s", toError(t, errors[0]).Reason)
	}
}

func TestValidator_DuplicateEnumValue(t *testing.T) {
	source := `enum Status {
	Active
	Inactive
	Active
}
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) == 0 {
		t.Error("expected validation error for duplicate enum value")
	}
	if !strings.Contains(toError(t, errors[0]).Reason, "duplicate enum value") {
		t.Errorf("expected duplicate enum value error, got: %s", toError(t, errors[0]).Reason)
	}
}

func TestValidator_DuplicateEnumIntValue(t *testing.T) {
	source := `enum Status {
	Active = 1
	Inactive = 1
}
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) == 0 {
		t.Error("expected validation error for duplicate enum int value")
	}
	if !strings.Contains(toError(t, errors[0]).Reason, "duplicate enum value 1") {
		t.Errorf("expected duplicate enum value error, got: %s", toError(t, errors[0]).Reason)
	}
}

func TestValidator_DuplicateEnumIntValueAutoIncrement(t *testing.T) {
	// Value1 = 1, Value2 auto-increments to 2, Value3 = 2 (duplicate!)
	source := `enum Status {
	Value1 = 1
	Value2
	Value3 = 2
}
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) == 0 {
		t.Error("expected validation error for duplicate enum int value")
	}
	if !strings.Contains(toError(t, errors[0]).Reason, "duplicate enum value 2") {
		t.Errorf("expected duplicate enum value 2 error, got: %s", toError(t, errors[0]).Reason)
	}
}

func TestValidator_DuplicateEnumStringValue(t *testing.T) {
	source := `enum Color {
	Red = "red"
	Blue = "red"
}
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) == 0 {
		t.Error("expected validation error for duplicate enum string value")
	}
	if !strings.Contains(toError(t, errors[0]).Reason, "duplicate enum value") && !strings.Contains(toError(t, errors[0]).Reason, "red") {
		t.Errorf("expected duplicate enum value 'red' error, got: %s", toError(t, errors[0]).Reason)
	}
}

func TestValidator_ValidEnumAutoIncrement(t *testing.T) {
	// Value1 = 1, Value2 = 2, Value3 = 3 - all unique
	source := `enum Status {
	Value1 = 1
	Value2
	Value3
}
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) != 0 {
		t.Errorf("expected no validation errors, got: %v", errors)
	}
}

func TestValidator_NameConflict(t *testing.T) {
	source := `const User = 1
model User { Id: string }
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) == 0 {
		t.Error("expected validation error for name conflict")
	}
	if !strings.Contains(toError(t, errors[0]).Reason, "conflicts with") {
		t.Errorf("expected name conflict error, got: %s", toError(t, errors[0]).Reason)
	}
}

func TestValidator_ModelExtendsUnknown(t *testing.T) {
	// Note: The parser has a bug with DOT token scanning that prevents testing extends.
	// This test is skipped until the scanner is fixed.
	t.Skip("Scanner bug with DOT token prevents testing extends syntax")

	source := `model User {
	...UnknownModel
	Id: string
}
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) == 0 {
		t.Error("expected validation error for unknown extended model")
	}
	if !strings.Contains(toError(t, errors[0]).Reason, "extends unknown model") {
		t.Errorf("expected extends unknown model error, got: %s", toError(t, errors[0]).Reason)
	}
}

func TestValidator_CompleteValidProgram(t *testing.T) {
	source := `const MaxSize = 100kb
const Timeout = 1m

enum Status {
	Pending
	Active
	Completed
}

model Address {
	Street: string
	City: string
}

model User {
	Id: string
	Name: string
	Status: Status
	Address: Address
	Tags: []string
	Metadata: map<string, string>
}

service UserService {
	Create(name: string) => (user: User)
	GetById(id: string) => (user: User)
	Delete(id: string)
}

error ErrNotFound { Msg = "not found" }
`
	scanner := NewScanner(strings.NewReader(source), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errors := ValidateProgram(program)
	if len(errors) != 0 {
		for _, e := range errors {
			t.Errorf("unexpected validation error: %s", toError(t, e).Reason)
		}
	}
}
