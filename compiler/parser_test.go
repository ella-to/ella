package compiler_test

import (
	"fmt"
	"strings"
	"testing"

	"ella.to/ella/compiler"
)

func runParserTest(t *testing.T, input string, output string) {
	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(input), "test.ella"))

	prog, err := parser.Parse()
	if err != nil {
		t.Errorf("unexpected error during parsing: %v", err)
		return
	}

	var sb strings.Builder

	for _, node := range prog.Nodes {
		sb.WriteString(node.String())
		sb.WriteString("\n")
	}

	gotOutput := strings.TrimSpace(sb.String())
	expectedOutput := strings.TrimSpace(output)

	if gotOutput != expectedOutput {
		t.Errorf("incorrect output")
		fmt.Println(gotOutput)
	}
}

func TestConstParser(t *testing.T) {
	input := `const PI = 3.14`
	output := `const PI = 3.14`

	runParserTest(t, input, output)
}

func TestConstStringDoubleQuoteParser(t *testing.T) {
	input := `const TopicUserCreated = "jetdrive.user.created"`
	output := `const TopicUserCreated = "jetdrive.user.created"`

	runParserTest(t, input, output)
}

func TestConstStringSingleQuoteParser(t *testing.T) {
	input := `const TopicUserCreated = 'jetdrive.user.created'`
	output := `const TopicUserCreated = 'jetdrive.user.created'`

	runParserTest(t, input, output)
}

func TestConstStringBacktickParser(t *testing.T) {
	input := "const TopicUserCreated = `jetdrive.user.created`"
	output := "const TopicUserCreated = `jetdrive.user.created`"

	runParserTest(t, input, output)
}

func TestConstStringNoQuoteParserError(t *testing.T) {
	// This test verifies that an identifier followed by DOT produces an error
	// (user probably forgot to quote the string)
	input := `const TopicDeviceCreated = jetdrive.device.created`

	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(input), "test.ella"))
	_, err := parser.Parse()

	if err == nil {
		t.Error("expected error for identifier followed by DOT, got none")
		return
	}

	if !strings.Contains(err.Error(), "did you mean to use a string") {
		t.Errorf("expected helpful error message about missing quotes, got: %v", err)
	}
}

func TestMultipleStringConstParser(t *testing.T) {
	input := `
const TopicUserCreated = "jetdrive.user.created"
const TopicUserStatusUpdated = "jetdrive.user.status.updated"
const TopicUserDeleted = "jetdrive.user.deleted"
`
	output := `
const TopicUserCreated = "jetdrive.user.created"
const TopicUserStatusUpdated = "jetdrive.user.status.updated"
const TopicUserDeleted = "jetdrive.user.deleted"
`

	runParserTest(t, input, output)
}

func TestEnumParser(t *testing.T) {
	input := `
enum Color {
	RED
	GREEN
	BLUE
}	
`

	output := `
enum Color {
	RED
	GREEN
	BLUE
}

`

	runParserTest(t, input, output)
}

func TestModelParser(t *testing.T) {
	input := `
model Person {
	name: string


	
	age: number
	isEmployed: bool
}
`

	output := `
model Person {
	name: string
	age: number
	isEmployed: bool
}
`

	runParserTest(t, input, output)
}

func TestModelParser_OptionalField(t *testing.T) {
	input := `
model Person {
	name?: string
	age: number
}
`

	output := `
model Person {
	name?: string
	age: number
}
`

	runParserTest(t, input, output)
}

func TestCommentParser(t *testing.T) {
	input := `
# This is a comment
const PI = 3.14 # Inline comment
`
	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(input), "test.ella"))
	prog, err := parser.Parse()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(prog.Comments) != 2 {
		t.Errorf("expected 2 comments, got %d", len(prog.Comments))
	}

	if prog.Comments[0].Lit != "# This is a comment" {
		t.Errorf("unexpected comment content: %s", prog.Comments[0].Lit)
	}
}
