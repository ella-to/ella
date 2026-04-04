package compiler_test

import (
	"fmt"
	"strings"
	"testing"

	"ella.to/ella/compiler"
)

type (
	Pos   = compiler.Pos
	Token = compiler.Token
)

func printTokens(tokens []*compiler.Token) {
	var sb strings.Builder

	//{Type: compiler.COMMENT, Pos: Pos{Offset: 0, Line: 1, Column: 1}, Lit: "# This is a comment"}
	for _, tok := range tokens {
		sb.WriteString("{Type: compiler.")
		sb.WriteString(tok.Type.String())
		sb.WriteString(", Pos: Pos{Offset: ")
		sb.WriteString(fmt.Sprintf("%d", tok.Pos.Offset))
		sb.WriteString(", Line: ")
		sb.WriteString(fmt.Sprintf("%d", tok.Pos.Line))
		sb.WriteString(", Column: ")
		sb.WriteString(fmt.Sprintf("%d", tok.Pos.Column))
		sb.WriteString("}, Lit: \"")
		sb.WriteString(tok.Lit)
		sb.WriteString("\"},\n")
	}

	println(sb.String())
}

func runTestScanner(t *testing.T, input string, expected []*compiler.Token) {
	scanner := compiler.NewScanner(strings.NewReader(input), "")
	tokens := make([]*compiler.Token, 0)
	for {
		tok, err := scanner.Scan()
		if err != nil {
			t.Fatalf("scan error: %v", err)
		}
		tokens = append(tokens, tok)

		if tok.Type == compiler.EOF {
			break
		}
	}

	var hasError bool

	if len(tokens) != len(expected) {
		hasError = true
		t.Errorf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, tok := range tokens {
		if i >= len(expected) {
			break
		}
		exp := expected[i]

		if tok.Type != exp.Type {
			hasError = true
			t.Errorf("test %d: expected token type %v, got %v", i, exp.Type, tok.Type)
		}

		if tok.Lit != exp.Lit {
			hasError = true
			t.Errorf("test %d: expected literal %q, got %q", i, exp.Lit, tok.Lit)
		}

		if tok.Pos != exp.Pos {
			hasError = true
			t.Errorf("test %d: expected position %+v, got %+v", i, exp.Pos, tok.Pos)
		}

	}

	if hasError {
		printTokens(tokens)
	}
}

func TestScanComment(t *testing.T) {
	input := `# This is a comment`

	expected := []*compiler.Token{
		{Type: compiler.COMMENT, Pos: Pos{Offset: 0, Line: 1, Column: 1}, Lit: "# This is a comment"},
		{Type: compiler.EOF, Pos: Pos{Offset: 18, Line: 1, Column: 19}, Lit: ""},
	}

	runTestScanner(t, input, expected)
}

func TestScanConstantValue(t *testing.T) {
	input := `3.14             `

	expected := []*compiler.Token{
		{Type: compiler.CONST_NUMBER, Pos: Pos{Offset: 0, Line: 1, Column: 1}, Lit: "3.14"},
		{Type: compiler.EOF, Pos: Pos{Offset: 16, Line: 1, Column: 17}, Lit: ""},
	}

	runTestScanner(t, input, expected)
}

func TestScanConst(t *testing.T) {
	input := `const pi = 3.14`

	expected := []*compiler.Token{
		{Type: compiler.CONST, Pos: Pos{Offset: 0, Line: 1, Column: 1}, Lit: "const"},
		{Type: compiler.IDENTIFIER, Pos: Pos{Offset: 6, Line: 1, Column: 7}, Lit: "pi"},
		{Type: compiler.EQUAL, Pos: Pos{Offset: 9, Line: 1, Column: 10}, Lit: "="},
		{Type: compiler.CONST_NUMBER, Pos: Pos{Offset: 11, Line: 1, Column: 12}, Lit: "3.14"},
		{Type: compiler.EOF, Pos: Pos{Offset: 14, Line: 1, Column: 15}, Lit: ""},
	}

	runTestScanner(t, input, expected)
}

func TestScanModel(t *testing.T) {
	input := `
model Circle {
	radius: int6
}
	`

	expected := []*compiler.Token{
		{Type: compiler.MODEL, Pos: Pos{Offset: 1, Line: 2, Column: 1}, Lit: "model"},
		{Type: compiler.IDENTIFIER, Pos: Pos{Offset: 7, Line: 2, Column: 7}, Lit: "Circle"},
		{Type: compiler.OPEN_CURLY, Pos: Pos{Offset: 14, Line: 2, Column: 14}, Lit: "{"},
		{Type: compiler.IDENTIFIER, Pos: Pos{Offset: 17, Line: 3, Column: 2}, Lit: "radius"},
		{Type: compiler.COLON, Pos: Pos{Offset: 23, Line: 3, Column: 8}, Lit: ":"},
		{Type: compiler.IDENTIFIER, Pos: Pos{Offset: 25, Line: 3, Column: 10}, Lit: "int6"},
		{Type: compiler.CLOSE_CURLY, Pos: Pos{Offset: 30, Line: 4, Column: 1}, Lit: "}"},
		{Type: compiler.EOF, Pos: Pos{Offset: 32, Line: 5, Column: 1}, Lit: ""},
	}

	runTestScanner(t, input, expected)
}

func TestRandomStuff(t *testing.T) {
	testCases := []struct {
		input    string
		expected []*compiler.Token
	}{
		{
			input: `const MaxLogoAssetSize = 100kb`,
			expected: []*compiler.Token{
				{Type: compiler.CONST, Pos: Pos{Offset: 0, Line: 1, Column: 1}, Lit: "const"},
				{Type: compiler.IDENTIFIER, Pos: Pos{Offset: 6, Line: 1, Column: 7}, Lit: "MaxLogoAssetSize"},
				{Type: compiler.EQUAL, Pos: Pos{Offset: 23, Line: 1, Column: 24}, Lit: "="},
				{Type: compiler.CONST_NUMBER, Pos: Pos{Offset: 25, Line: 1, Column: 26}, Lit: "100"},
				{Type: compiler.IDENTIFIER, Pos: Pos{Offset: 28, Line: 1, Column: 29}, Lit: "kb"},
				{Type: compiler.EOF, Pos: Pos{Offset: 29, Line: 1, Column: 30}, Lit: ""},
			},
		},
		{
			input: `
			model Business {
				Id: string {
					json = false
				}
			}`,
			expected: []*compiler.Token{
				{Type: compiler.MODEL, Pos: Pos{Offset: 4, Line: 2, Column: 4}, Lit: "model"},
				{Type: compiler.IDENTIFIER, Pos: Pos{Offset: 10, Line: 2, Column: 10}, Lit: "Business"},
				{Type: compiler.OPEN_CURLY, Pos: Pos{Offset: 19, Line: 2, Column: 19}, Lit: "{"},
				{Type: compiler.IDENTIFIER, Pos: Pos{Offset: 25, Line: 3, Column: 5}, Lit: "Id"},
				{Type: compiler.COLON, Pos: Pos{Offset: 27, Line: 3, Column: 7}, Lit: ":"},
				{Type: compiler.STRING, Pos: Pos{Offset: 29, Line: 3, Column: 9}, Lit: "string"},
				{Type: compiler.OPEN_CURLY, Pos: Pos{Offset: 36, Line: 3, Column: 16}, Lit: "{"},
				{Type: compiler.IDENTIFIER, Pos: Pos{Offset: 43, Line: 4, Column: 6}, Lit: "json"},
				{Type: compiler.EQUAL, Pos: Pos{Offset: 48, Line: 4, Column: 11}, Lit: "="},
				{Type: compiler.IDENTIFIER, Pos: Pos{Offset: 50, Line: 4, Column: 13}, Lit: "false"},
				{Type: compiler.CLOSE_CURLY, Pos: Pos{Offset: 60, Line: 5, Column: 5}, Lit: "}"},
				{Type: compiler.CLOSE_CURLY, Pos: Pos{Offset: 65, Line: 6, Column: 4}, Lit: "}"},
				{Type: compiler.EOF, Pos: Pos{Offset: 65, Line: 6, Column: 4}, Lit: ""},
			},
		},
	}

	for _, tc := range testCases {
		runTestScanner(t, tc.input, tc.expected)
	}
}
