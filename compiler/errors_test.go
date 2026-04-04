package compiler_test

import (
	"strings"
	"testing"

	"ella.to/ella/compiler"
)

func TestErrorDisplayMissingCurly(t *testing.T) {
	input := `
const MaxSize = 100kb

model User
    Id: string
    Name: string
}

service UserService {
    GetUser (id: string) => (user: User)
}
`

	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(input), "test.ella"))
	_, err := parser.Parse()

	if err == nil {
		t.Fatal("expected an error but got none")
	}

	ed := compiler.NewErrorDisplay(input, "test.ella")
	formatted := ed.FormatErrorPlain(err)

	// Verify the error output contains expected parts
	if !strings.Contains(formatted, "error:") {
		t.Errorf("expected 'error:' in output, got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "test.ella") {
		t.Errorf("expected filename in output, got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "^") {
		t.Errorf("expected error pointer '^' in output, got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "model User") {
		t.Errorf("expected context line 'model User' in output, got:\n%s", formatted)
	}

	// Print for visual inspection
	t.Logf("Error display output:\n%s", formatted)
}

func TestErrorDisplayUnclosedString(t *testing.T) {
	input := `
const AppName = "MyApp
const Version = "1.0.0"
`

	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(input), "test.ella"))
	_, err := parser.Parse()

	if err == nil {
		t.Fatal("expected an error but got none")
	}

	ed := compiler.NewErrorDisplay(input, "config.ella")
	formatted := ed.FormatErrorPlain(err)

	if !strings.Contains(formatted, "error:") {
		t.Errorf("expected 'error:' in output, got:\n%s", formatted)
	}

	t.Logf("Error display output:\n%s", formatted)
}

func TestErrorDisplayMissingType(t *testing.T) {
	input := `
model Product {
    Id: string
    Name:
    Price: float64
}
`

	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(input), "test.ella"))
	_, err := parser.Parse()

	if err == nil {
		t.Fatal("expected an error but got none")
	}

	ed := compiler.NewErrorDisplay(input, "product.ella")
	formatted := ed.FormatErrorPlain(err)

	// Should show context around the error
	if !strings.Contains(formatted, "Id: string") {
		t.Errorf("expected context line 'Id: string' in output, got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "Price: float64") {
		t.Errorf("expected context line 'Price: float64' in output, got:\n%s", formatted)
	}

	t.Logf("Error display output:\n%s", formatted)
}

func TestErrorDisplayServiceMethod(t *testing.T) {
	input := `
model User {
    Id: string
}

service UserService {
    GetUser id: string) => (user: User)
    DeleteUser (id: string)
}
`

	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(input), "test.ella"))
	_, err := parser.Parse()

	if err == nil {
		t.Fatal("expected an error but got none")
	}

	ed := compiler.NewErrorDisplay(input, "service.ella")
	formatted := ed.FormatErrorPlain(err)

	t.Logf("Error display output:\n%s", formatted)
}

func TestErrorDisplayWithColors(t *testing.T) {
	input := `
enum Status {
    Active
    Inactive

model User {
    Id: string
}
`

	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(input), "test.ella"))
	_, err := parser.Parse()

	if err == nil {
		t.Fatal("expected an error but got none")
	}

	ed := compiler.NewErrorDisplay(input, "status.ella")
	formatted := ed.FormatError(err)

	// Check that ANSI codes are present
	if !strings.Contains(formatted, "\033[") {
		t.Errorf("expected ANSI color codes in output, got:\n%s", formatted)
	}

	t.Logf("Error display with colors:\n%s", formatted)
}

func TestErrorDisplayNoFilename(t *testing.T) {
	input := `
const X = 
`

	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(input), "test.ella"))
	_, err := parser.Parse()

	if err == nil {
		t.Fatal("expected an error but got none")
	}

	ed := compiler.NewErrorDisplay(input, "")
	formatted := ed.FormatErrorPlain(err)

	// Should show "line X, column Y" instead of filename
	if !strings.Contains(formatted, "line") {
		t.Errorf("expected 'line' in output when no filename, got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "column") {
		t.Errorf("expected 'column' in output when no filename, got:\n%s", formatted)
	}

	t.Logf("Error display output:\n%s", formatted)
}

func TestErrorDisplayErrorDeclaration(t *testing.T) {
	input := `
error ErrNotFound { Msg = }
`

	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(input), "test.ella"))
	_, err := parser.Parse()

	if err == nil {
		t.Fatal("expected an error but got none")
	}

	ed := compiler.NewErrorDisplay(input, "errors.ella")
	formatted := ed.FormatErrorPlain(err)

	t.Logf("Error display output:\n%s", formatted)
}
