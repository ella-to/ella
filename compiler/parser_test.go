package compiler_test

import (
	"fmt"
	"strings"
	"testing"

	"ella.to/ella/compiler"
)

// withModule prefixes a test source with the now-required module declaration.
// When the source already begins with a newline the module goes on that first
// (previously blank) line so content line numbers are preserved; otherwise it
// is added on a new first line.
func withModule(src string) string {
	if strings.HasPrefix(src, "\n") {
		return "module test" + src
	}
	return "module test\n" + src
}

// stripModule removes the leading `module ...` line (and the blank line after
// it) that Format now emits, so tests can keep asserting on the rest.
func stripModule(formatted string) string {
	if i := strings.IndexByte(formatted, '\n'); i >= 0 && strings.HasPrefix(formatted, "module ") {
		return strings.TrimLeft(formatted[i+1:], "\n")
	}
	return formatted
}

func runParserTest(t *testing.T, input string, output string) {
	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(withModule(input)), "test.ella"))

	prog, err := parser.Parse()
	if err != nil {
		t.Errorf("unexpected error during parsing: %v", err)
		return
	}

	var sb strings.Builder

	for _, node := range prog.Nodes {
		// The injected module declaration is not part of what these tests assert.
		if _, ok := node.(*compiler.DeclModule); ok {
			continue
		}
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

	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(withModule(input)), "test.ella"))
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

func TestServiceParser_OptionalArguments(t *testing.T) {
	input := `
service UserService {
	List (query: string, limit?: int64, tags?: []string) => (ids: []string)
}
`

	output := `
service UserService {
	List (query: string, limit?: int64, tags?: []string) => (ids: []string)
}
`

	runParserTest(t, input, output)
}

func TestServiceParser_OptionalReturnRejected(t *testing.T) {
	input := `
service UserService {
	List (query: string) => (ids?: []string)
}
`

	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(withModule(input)), "test.ella"))
	_, err := parser.Parse()

	if err == nil {
		t.Error("expected error for optional marker in return values, got none")
		return
	}

	if !strings.Contains(err.Error(), "not allowed in return values") {
		t.Errorf("expected error about optional return values, got: %v", err)
	}
}

func TestCommentParser(t *testing.T) {
	input := `
# This is a comment
const PI = 3.14 # Inline comment
`
	parser := compiler.NewParser(compiler.NewScanner(strings.NewReader(withModule(input)), "test.ella"))
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
