package compiler_test

import (
	"strings"
	"testing"

	"ella.to/ella/compiler"
)

func TestFormat(t *testing.T) {
	input := `
const MaxLogoAssetSize = 100kb
const TimeoutLogoAsset = 1m
const TopicBusinessCreated = "rentify.business.created"
const TopicBusinessUpdated = "rentify.business.updated"
const TopicBusinessDeleted = "rentify.business.deleted"

model Business {
    Id: string
    Name: string
    Attributes: []Attribute
    CreatedOn: string
    UpdatedOn: string
}

# This is comment
service HttpBusinessService { # this is another comment
    Create (name: string, attributes: []Attribute) => (result: Business)
    Update (business: Business) => (result: Business)
    Delete (businessId: string)
    GetById (businessId: string) => (result: Business)
    SearchByName (name: string, size: int64, lastId: string) => (results: []Business)
    GetLogoUploadToken () => (token: string)
}

service RpcBusinessService {
    Create (name: string, attributes: []Attribute) => (result: Business)
    Update (business: Business) => (result: Business)
    Delete (businessId: string)
    GetById (businessId: string) => (result: Business)
    SearchByName (name: string, size: int64, lastId: string) => (results: []Business)
    GetByName (name: string) => (result: Business)
}
`

	expected := `const MaxLogoAssetSize = 100kb
const TimeoutLogoAsset = 1m
const TopicBusinessCreated = "rentify.business.created"
const TopicBusinessUpdated = "rentify.business.updated"
const TopicBusinessDeleted = "rentify.business.deleted"

model Business {
	Id: string
	Name: string
	Attributes: []Attribute
	CreatedOn: string
	UpdatedOn: string
}

# This is comment
service HttpBusinessService { # this is another comment
	Create (name: string, attributes: []Attribute) => (result: Business)
	Update (business: Business) => (result: Business)
	Delete (businessId: string)
	GetById (businessId: string) => (result: Business)
	SearchByName (name: string, size: int64, lastId: string) => (results: []Business)
	GetLogoUploadToken () => (token: string)
}

service RpcBusinessService {
	Create (name: string, attributes: []Attribute) => (result: Business)
	Update (business: Business) => (result: Business)
	Delete (businessId: string)
	GetById (businessId: string) => (result: Business)
	SearchByName (name: string, size: int64, lastId: string) => (results: []Business)
	GetByName (name: string) => (result: Business)
}`

	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(input), "test.ella"))
	prog, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	formatted := compiler.Format(prog)
	if formatted != strings.TrimSpace(expected) {
		t.Errorf("formatted output does not match expected.\nExpected:\n%s\nGot:\n%s", expected, formatted)
	}
}

func TestFormatWithErrors(t *testing.T) {
	input := `
const MaxSize = 100kb
const Timeout = 1m

model User {
    Id: string
    Name: string
}

model Product {
    Id: string
    Title: string
    Price: float64
}

service UserService {
    Create (name: string) => (user: User)
    Delete (id: string)
}

error ErrUserNotFound { Msg = "user not found" }
error ErrUserAlreadyExists { Msg = "user already exists" }
error ErrInvalidInput { Msg = "invalid input provided" }
`

	expected := `const MaxSize = 100kb
const Timeout = 1m

model User {
	Id: string
	Name: string
}

model Product {
	Id: string
	Title: string
	Price: float64
}

service UserService {
	Create (name: string) => (user: User)
	Delete (id: string)
}

error ErrUserNotFound { Msg = "user not found" }
error ErrUserAlreadyExists { Msg = "user already exists" }
error ErrInvalidInput { Msg = "invalid input provided" }`

	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(input), "test.ella"))
	prog, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	formatted := compiler.Format(prog)
	if formatted != strings.TrimSpace(expected) {
		t.Errorf("formatted output does not match expected.\nExpected:\n%s\nGot:\n%s", expected, formatted)
	}
}

func TestFormatCommentsInsideBlocks(t *testing.T) {
	input := `
model Person {
    # This is the person's ID
    Id: string
    # Full name of the person
    Name: string
    # Age in years
    Age: int32
}

enum Status {
    # Active status
    Active
    # Inactive status
    Inactive
    # Pending review
    Pending
}
`

	expected := `enum Status {
	# Active status
	Active
	# Inactive status
	Inactive
	# Pending review
	Pending
}

model Person {
	# This is the person's ID
	Id: string
	# Full name of the person
	Name: string
	# Age in years
	Age: int32
}`

	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(input), "test.ella"))
	prog, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	formatted := compiler.Format(prog)
	if formatted != strings.TrimSpace(expected) {
		t.Errorf("formatted output does not match expected.\nExpected:\n%s\nGot:\n%s", expected, formatted)
	}
}

func TestFormatMixedWithComments(t *testing.T) {
	input := `
# Application constants
const AppName = "MyApp"
const Version = "1.0.0"
# Timeout settings
const DefaultTimeout = 30s

# User model represents a system user
model User {
    Id: string
    Email: string
}

# Admin model extends user with admin privileges
model Admin {
    Id: string
    Role: string
}

# User management service
service UserService { # handles all user operations
    GetUser (id: string) => (user: User)
}

# Error definitions
error ErrNotFound { Msg = "resource not found" }
# Authentication errors
error ErrUnauthorized { Msg = "unauthorized access" }
`

	expected := `# Application constants
const AppName = "MyApp"
const Version = "1.0.0"
# Timeout settings
const DefaultTimeout = 30s

# User model represents a system user
model User {
	Id: string
	Email: string
}

# Admin model extends user with admin privileges
model Admin {
	Id: string
	Role: string
}

# User management service
service UserService { # handles all user operations
	GetUser (id: string) => (user: User)
}

# Error definitions
error ErrNotFound { Msg = "resource not found" }
# Authentication errors
error ErrUnauthorized { Msg = "unauthorized access" }`

	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(input), "test.ella"))
	prog, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	formatted := compiler.Format(prog)
	if formatted != strings.TrimSpace(expected) {
		t.Errorf("formatted output does not match expected.\nExpected:\n%s\nGot:\n%s", expected, formatted)
	}
}

func TestFormatEmptyBlocks(t *testing.T) {
	input := `
model Empty {
}

service EmptyService {
}

enum EmptyEnum {
}
`

	expected := `enum EmptyEnum {
}

model Empty {
}

service EmptyService {
}`

	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(input), "test.ella"))
	prog, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	formatted := compiler.Format(prog)
	if formatted != strings.TrimSpace(expected) {
		t.Errorf("formatted output does not match expected.\nExpected:\n%s\nGot:\n%s", expected, formatted)
	}
}

func TestFormatComplexTypes(t *testing.T) {
	input := `
model ComplexModel {
    Tags: []string
    Metadata: map<string, string>
    NestedArray: [][]int32
    Items: []Item
}

service DataService {
    GetAll () => (items: []Item, total: int64)
    GetMap () => (data: map<string, Item>)
    Process (input: []string, options: map<string, bool>) => (result: string)
}
`

	expected := `model ComplexModel {
	Tags: []string
	Metadata: map<string, string>
	NestedArray: [][]int32
	Items: []Item
}

service DataService {
	GetAll () => (items: []Item, total: int64)
	GetMap () => (data: map<string, Item>)
	Process (input: []string, options: map<string, bool>) => (result: string)
}`

	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(input), "test.ella"))
	prog, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	formatted := compiler.Format(prog)
	if formatted != strings.TrimSpace(expected) {
		t.Errorf("formatted output does not match expected.\nExpected:\n%s\nGot:\n%s", expected, formatted)
	}
}

func TestFormatOnlyConsts(t *testing.T) {
	input := `
const A = 1
const B = 2
const C = 3
const D = "hello"
const E = true
`

	expected := `const A = 1
const B = 2
const C = 3
const D = "hello"
const E = true`

	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(input), "test.ella"))
	prog, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	formatted := compiler.Format(prog)
	if formatted != strings.TrimSpace(expected) {
		t.Errorf("formatted output does not match expected.\nExpected:\n%s\nGot:\n%s", expected, formatted)
	}
}

func TestFormatOnlyErrors(t *testing.T) {
	input := `
error ErrOne { Msg = "error one" }
error ErrTwo { Msg = "error two" }
error ErrThree { Msg = "error three" }
`

	expected := `error ErrOne { Msg = "error one" }
error ErrTwo { Msg = "error two" }
error ErrThree { Msg = "error three" }`

	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(input), "test.ella"))
	prog, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	formatted := compiler.Format(prog)
	if formatted != strings.TrimSpace(expected) {
		t.Errorf("formatted output does not match expected.\nExpected:\n%s\nGot:\n%s", expected, formatted)
	}
}

func TestFormatMultipleEnumsSeparatedByBlankLine(t *testing.T) {
	input := `
enum Status {
	Active
}
enum Role {
	Admin
}
`

	expected := `enum Status {
	Active
}

enum Role {
	Admin
}`

	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(input), "test.ella"))
	prog, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	formatted := compiler.Format(prog)
	if formatted != strings.TrimSpace(expected) {
		t.Errorf("formatted output does not match expected.\nExpected:\n%s\nGot:\n%s", expected, formatted)
	}
}
