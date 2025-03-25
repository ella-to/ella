package scanner

import (
	"fmt"
	"strings"
	"testing"

	"ella.to/ella/internal/compiler/token"
	"github.com/stretchr/testify/assert"
)

type Tokens []token.Token

type TestCase struct {
	input  string
	output Tokens
	skip   bool
}

type TestCases []TestCase

func (t Tokens) String() string {
	var sb strings.Builder
	sb.WriteString("\n")
	for i := range t {
		sb.WriteString(fmt.Sprintf("{Type: token.%s, Start: %d, End: %d, Value: \"%s\"},\n", t[i].Type, t[i].Start, t[i].End, t[i].Value))
	}
	return sb.String()
}

func runTestCase(t *testing.T, target int, initState State, testCases TestCases) {
	if target > -1 && target < len(testCases) {
		testCases = TestCases{testCases[target]}
	}

	for i, tc := range testCases {
		if tc.skip {
			continue
		}
		output := make(Tokens, 0)
		emitter := token.EmitterFunc(func(token *token.Token) {
			output = append(output, *token)
		})

		Start(emitter, initState, tc.input)
		assert.Equal(t, tc.output, output, "Failed scanner at %d: %s", i, output)
	}
}

func TestLex(t *testing.T) {
	runTestCase(t, -1, Lex, TestCases{
		{
			input: `model User {
				id: int64
				name?: string
			}`,
			output: Tokens{
				{Type: token.Model, Start: 0, End: 5, Value: "model"},
				{Type: token.Identifier, Start: 6, End: 10, Value: "User"},
				{Type: token.OpenCurly, Start: 11, End: 12, Value: "{"},
				{Type: token.Identifier, Start: 17, End: 19, Value: "id"},
				{Type: token.Colon, Start: 19, End: 20, Value: ":"},
				{Type: token.Int64, Start: 21, End: 26, Value: "int64"},
				{Type: token.Identifier, Start: 31, End: 35, Value: "name"},
				{Type: token.Optional, Start: 35, End: 36, Value: "?"},
				{Type: token.Colon, Start: 36, End: 37, Value: ":"},
				{Type: token.String, Start: 38, End: 44, Value: "string"},
				{Type: token.CloseCurly, Start: 48, End: 49, Value: "}"},
				{Type: token.EOF, Start: 49, End: 49, Value: ""},
			},
		},
		{
			input: `service HttpFoo {
				GetAssetFile(assetId: string) => (result: stream []byte)
			}`,
			output: Tokens{
				{Type: token.Service, Start: 0, End: 7, Value: "service"},
				{Type: token.Identifier, Start: 8, End: 15, Value: "HttpFoo"},
				{Type: token.OpenCurly, Start: 16, End: 17, Value: "{"},
				{Type: token.Identifier, Start: 22, End: 34, Value: "GetAssetFile"},
				{Type: token.OpenParen, Start: 34, End: 35, Value: "("},
				{Type: token.Identifier, Start: 35, End: 42, Value: "assetId"},
				{Type: token.Colon, Start: 42, End: 43, Value: ":"},
				{Type: token.String, Start: 44, End: 50, Value: "string"},
				{Type: token.CloseParen, Start: 50, End: 51, Value: ")"},
				{Type: token.Return, Start: 52, End: 54, Value: "=>"},
				{Type: token.OpenParen, Start: 55, End: 56, Value: "("},
				{Type: token.Identifier, Start: 56, End: 62, Value: "result"},
				{Type: token.Colon, Start: 62, End: 63, Value: ":"},
				{Type: token.Stream, Start: 64, End: 70, Value: "stream"},
				{Type: token.Array, Start: 71, End: 73, Value: "[]"},
				{Type: token.Byte, Start: 73, End: 77, Value: "byte"},
				{Type: token.CloseParen, Start: 77, End: 78, Value: ")"},
				{Type: token.CloseCurly, Start: 82, End: 83, Value: "}"},
				{Type: token.EOF, Start: 83, End: 83, Value: ""},
			},
		},
		{
			input: `service RpcFoo {
				GetFoo() => (value: int64) {
					Required
					A = 1mb
					B = 100h
				}
			}`,
			output: Tokens{
				{Type: token.Service, Start: 0, End: 7, Value: "service"},
				{Type: token.Identifier, Start: 8, End: 14, Value: "RpcFoo"},
				{Type: token.OpenCurly, Start: 15, End: 16, Value: "{"},
				{Type: token.Identifier, Start: 21, End: 27, Value: "GetFoo"},
				{Type: token.OpenParen, Start: 27, End: 28, Value: "("},
				{Type: token.CloseParen, Start: 28, End: 29, Value: ")"},
				{Type: token.Return, Start: 30, End: 32, Value: "=>"},
				{Type: token.OpenParen, Start: 33, End: 34, Value: "("},
				{Type: token.Identifier, Start: 34, End: 39, Value: "value"},
				{Type: token.Colon, Start: 39, End: 40, Value: ":"},
				{Type: token.Int64, Start: 41, End: 46, Value: "int64"},
				{Type: token.CloseParen, Start: 46, End: 47, Value: ")"},
				{Type: token.OpenCurly, Start: 48, End: 49, Value: "{"},
				{Type: token.Identifier, Start: 55, End: 63, Value: "Required"},
				{Type: token.Identifier, Start: 69, End: 70, Value: "A"},
				{Type: token.Assign, Start: 71, End: 72, Value: "="},
				{Type: token.ConstBytes, Start: 73, End: 76, Value: "1mb"},
				{Type: token.Identifier, Start: 82, End: 83, Value: "B"},
				{Type: token.Assign, Start: 84, End: 85, Value: "="},
				{Type: token.ConstDuration, Start: 86, End: 90, Value: "100h"},
				{Type: token.CloseCurly, Start: 95, End: 96, Value: "}"},
				{Type: token.CloseCurly, Start: 100, End: 101, Value: "}"},
				{Type: token.EOF, Start: 101, End: 101, Value: ""},
			},
		},
		{
			input: `A = 1mb`,
			output: Tokens{
				{Type: token.Identifier, Start: 0, End: 1, Value: "A"},
				{Type: token.Assign, Start: 2, End: 3, Value: "="},
				{Type: token.ConstBytes, Start: 4, End: 7, Value: "1mb"},
				{Type: token.EOF, Start: 7, End: 7, Value: ""},
			},
		},
		{
			skip: true,
			input: `

			# this is a comment 1
			# this is another comment 2
			a = 1 # this is a comment 3
			# this is another comment 4

			message A {
				# this is a comment 5
				# this is another comment 6
				firstname: string
			}

			`,
			output: Tokens{
				{Type: token.Comment, Start: 9, End: 29, Value: " this is a comment 1"},
				{Type: token.Comment, Start: 34, End: 60, Value: " this is another comment 2"},
				{Type: token.Identifier, Start: 64, End: 65, Value: "a"},
				{Type: token.Assign, Start: 66, End: 67, Value: "="},
				{Type: token.ConstInt, Start: 68, End: 69, Value: "1"},
				{Type: token.Comment, Start: 71, End: 91, Value: " this is a comment 3"},
				{Type: token.Comment, Start: 96, End: 122, Value: " this is another comment 4"},
				{Type: token.Identifier, Start: 127, End: 134, Value: "message"},
				{Type: token.Identifier, Start: 135, End: 136, Value: "A"},
				{Type: token.OpenCurly, Start: 137, End: 138, Value: "{"},
				{Type: token.Comment, Start: 144, End: 164, Value: " this is a comment 5"},
				{Type: token.Comment, Start: 170, End: 196, Value: " this is another comment 6"},
				{Type: token.Identifier, Start: 201, End: 210, Value: "firstname"},
				{Type: token.Colon, Start: 210, End: 211, Value: ":"},
				{Type: token.String, Start: 212, End: 218, Value: "string"},
				{Type: token.CloseCurly, Start: 222, End: 223, Value: "}"},
				{Type: token.EOF, Start: 231, End: 231, Value: ""},
			},
		},
		{
			skip: true,
			input: `

			# This is a first comment
			a = 1 # this is the second comment
			# this is the third comment

			`,
			output: Tokens{
				{Type: token.Comment, Start: 6, End: 30, Value: " This is a first comment"},
				{Type: token.Identifier, Start: 34, End: 35, Value: "a"},
				{Type: token.Assign, Start: 36, End: 37, Value: "="},
				{Type: token.ConstInt, Start: 38, End: 39, Value: "1"},
				{Type: token.Comment, Start: 41, End: 68, Value: " this is the second comment"},
				{Type: token.Comment, Start: 73, End: 99, Value: " this is the third comment"},
				{Type: token.EOF, Start: 105, End: 105, Value: ""},
			},
		},
		{
			input: `ella = "1.0.0-b01"`,
			output: Tokens{
				{Type: token.Identifier, Start: 0, End: 4, Value: "ella"},
				{Type: token.Assign, Start: 5, End: 6, Value: "="},
				{Type: token.ConstStringDoubleQuote, Start: 8, End: 17, Value: "1.0.0-b01"},
				{Type: token.EOF, Start: 18, End: 18, Value: ""},
			},
		},
		{
			input: `message A {
				...B
				...C

				first: int64
			}`,
			output: Tokens{
				{Type: token.Identifier, Start: 0, End: 7, Value: "message"},
				{Type: token.Identifier, Start: 8, End: 9, Value: "A"},
				{Type: token.OpenCurly, Start: 10, End: 11, Value: "{"},
				{Type: token.Extend, Start: 16, End: 19, Value: "..."},
				{Type: token.Identifier, Start: 19, End: 20, Value: "B"},
				{Type: token.Extend, Start: 25, End: 28, Value: "..."},
				{Type: token.Identifier, Start: 28, End: 29, Value: "C"},
				{Type: token.Identifier, Start: 35, End: 40, Value: "first"},
				{Type: token.Colon, Start: 40, End: 41, Value: ":"},
				{Type: token.Int64, Start: 42, End: 47, Value: "int64"},
				{Type: token.CloseCurly, Start: 51, End: 52, Value: "}"},
				{Type: token.EOF, Start: 52, End: 52, Value: ""},
			},
		},
		{
			skip: true,
			input: `enum a int64 {
				one = 1 # comment
				two = 2# comment2
				three
			}`,
			output: Tokens{
				{Type: token.Enum, Start: 0, End: 4, Value: "enum"},
				{Type: token.Identifier, Start: 5, End: 6, Value: "a"},
				{Type: token.Int64, Start: 7, End: 12, Value: "int64"},
				{Type: token.OpenCurly, Start: 13, End: 14, Value: "{"},
				{Type: token.Identifier, Start: 19, End: 22, Value: "one"},
				{Type: token.Assign, Start: 23, End: 24, Value: "="},
				{Type: token.ConstInt, Start: 25, End: 26, Value: "1"},
				{Type: token.Comment, Start: 28, End: 36, Value: " comment"},
				{Type: token.Identifier, Start: 41, End: 44, Value: "two"},
				{Type: token.Assign, Start: 45, End: 46, Value: "="},
				{Type: token.ConstInt, Start: 47, End: 48, Value: "2"},
				{Type: token.Comment, Start: 49, End: 58, Value: " comment2"},
				{Type: token.Identifier, Start: 63, End: 68, Value: "three"},
				{Type: token.CloseCurly, Start: 72, End: 73, Value: "}"},
				{Type: token.EOF, Start: 73, End: 73, Value: ""},
			},
		},
		{
			input: `enum a int64 {
				one = 1
				two = 2
				three
			}`,
			output: Tokens{
				{Type: token.Enum, Start: 0, End: 4, Value: "enum"},
				{Type: token.Identifier, Start: 5, End: 6, Value: "a"},
				{Type: token.Int64, Start: 7, End: 12, Value: "int64"},
				{Type: token.OpenCurly, Start: 13, End: 14, Value: "{"},
				{Type: token.Identifier, Start: 19, End: 22, Value: "one"},
				{Type: token.Assign, Start: 23, End: 24, Value: "="},
				{Type: token.ConstInt, Start: 25, End: 26, Value: "1"},
				{Type: token.Identifier, Start: 31, End: 34, Value: "two"},
				{Type: token.Assign, Start: 35, End: 36, Value: "="},
				{Type: token.ConstInt, Start: 37, End: 38, Value: "2"},
				{Type: token.Identifier, Start: 43, End: 48, Value: "three"},
				{Type: token.CloseCurly, Start: 52, End: 53, Value: "}"},
				{Type: token.EOF, Start: 53, End: 53, Value: ""},
			},
		},
		{
			input: `enum a int64 {}`,
			output: Tokens{
				{Type: token.Enum, Start: 0, End: 4, Value: "enum"},
				{Type: token.Identifier, Start: 5, End: 6, Value: "a"},
				{Type: token.Int64, Start: 7, End: 12, Value: "int64"},
				{Type: token.OpenCurly, Start: 13, End: 14, Value: "{"},
				{Type: token.CloseCurly, Start: 14, End: 15, Value: "}"},
				{Type: token.EOF, Start: 15, End: 15, Value: ""},
			},
		},
		{
			input: `a=1`,
			output: Tokens{
				{Type: token.Identifier, Start: 0, End: 1, Value: "a"},
				{Type: token.Assign, Start: 1, End: 2, Value: "="},
				{Type: token.ConstInt, Start: 2, End: 3, Value: "1"},
				{Type: token.EOF, Start: 3, End: 3, Value: ""},
			},
		},
		{
			input: `

			a = 1.0

			message A {
				firstname: string {
					required
					pattern = "^[a-zA-Z]+$"
				}
			}

			service HttpMyService {
				GetUserById (id: int64) => (user: User) {
					method = "GET"
				}
			}

			`,
			output: Tokens{
				{Type: token.Identifier, Start: 5, End: 6, Value: "a"},
				{Type: token.Assign, Start: 7, End: 8, Value: "="},
				{Type: token.ConstFloat, Start: 9, End: 12, Value: "1.0"},
				{Type: token.Identifier, Start: 17, End: 24, Value: "message"},
				{Type: token.Identifier, Start: 25, End: 26, Value: "A"},
				{Type: token.OpenCurly, Start: 27, End: 28, Value: "{"},
				{Type: token.Identifier, Start: 33, End: 42, Value: "firstname"},
				{Type: token.Colon, Start: 42, End: 43, Value: ":"},
				{Type: token.String, Start: 44, End: 50, Value: "string"},
				{Type: token.OpenCurly, Start: 51, End: 52, Value: "{"},
				{Type: token.Identifier, Start: 58, End: 66, Value: "required"},
				{Type: token.Identifier, Start: 72, End: 79, Value: "pattern"},
				{Type: token.Assign, Start: 80, End: 81, Value: "="},
				{Type: token.ConstStringDoubleQuote, Start: 83, End: 94, Value: "^[a-zA-Z]+$"},
				{Type: token.CloseCurly, Start: 100, End: 101, Value: "}"},
				{Type: token.CloseCurly, Start: 105, End: 106, Value: "}"},
				{Type: token.Service, Start: 111, End: 118, Value: "service"},
				{Type: token.Identifier, Start: 119, End: 132, Value: "HttpMyService"},
				{Type: token.OpenCurly, Start: 133, End: 134, Value: "{"},
				{Type: token.Identifier, Start: 139, End: 150, Value: "GetUserById"},
				{Type: token.OpenParen, Start: 151, End: 152, Value: "("},
				{Type: token.Identifier, Start: 152, End: 154, Value: "id"},
				{Type: token.Colon, Start: 154, End: 155, Value: ":"},
				{Type: token.Int64, Start: 156, End: 161, Value: "int64"},
				{Type: token.CloseParen, Start: 161, End: 162, Value: ")"},
				{Type: token.Return, Start: 163, End: 165, Value: "=>"},
				{Type: token.OpenParen, Start: 166, End: 167, Value: "("},
				{Type: token.Identifier, Start: 167, End: 171, Value: "user"},
				{Type: token.Colon, Start: 171, End: 172, Value: ":"},
				{Type: token.Identifier, Start: 173, End: 177, Value: "User"},
				{Type: token.CloseParen, Start: 177, End: 178, Value: ")"},
				{Type: token.OpenCurly, Start: 179, End: 180, Value: "{"},
				{Type: token.Identifier, Start: 186, End: 192, Value: "method"},
				{Type: token.Assign, Start: 193, End: 194, Value: "="},
				{Type: token.ConstStringDoubleQuote, Start: 196, End: 199, Value: "GET"},
				{Type: token.CloseCurly, Start: 205, End: 206, Value: "}"},
				{Type: token.CloseCurly, Start: 210, End: 211, Value: "}"},
				{Type: token.EOF, Start: 216, End: 216, Value: ""},
			},
		},
		{
			input: `error ErrUserNotFound { Code = 1000 HttpStatus = NotFound Msg = "user not found" }`,
			output: Tokens{
				{Type: token.CustomError, Start: 0, End: 5, Value: "error"},
				{Type: token.Identifier, Start: 6, End: 21, Value: "ErrUserNotFound"},
				{Type: token.OpenCurly, Start: 22, End: 23, Value: "{"},
				{Type: token.Identifier, Start: 24, End: 28, Value: "Code"},
				{Type: token.Assign, Start: 29, End: 30, Value: "="},
				{Type: token.ConstInt, Start: 31, End: 35, Value: "1000"},
				{Type: token.Identifier, Start: 36, End: 46, Value: "HttpStatus"},
				{Type: token.Assign, Start: 47, End: 48, Value: "="},
				{Type: token.Identifier, Start: 49, End: 57, Value: "NotFound"},
				{Type: token.Identifier, Start: 58, End: 61, Value: "Msg"},
				{Type: token.Assign, Start: 62, End: 63, Value: "="},
				{Type: token.ConstStringDoubleQuote, Start: 65, End: 79, Value: "user not found"},
				{Type: token.CloseCurly, Start: 81, End: 82, Value: "}"},
				{Type: token.EOF, Start: 82, End: 82, Value: ""},
			},
		},
	})
}

func TestNumber(t *testing.T) {
	runTestCase(t, -1, Number,
		TestCases{
			{
				input: `1`,
				output: Tokens{
					{Type: token.ConstInt, Start: 0, End: 1, Value: "1"},
				},
			},
			{
				input: `1.0`,
				output: Tokens{
					{Type: token.ConstFloat, Start: 0, End: 3, Value: "1.0"},
				},
			},
			{
				input: `1.`,
				output: Tokens{
					{Type: token.Error, Start: 0, End: 2, Value: "expected digit after decimal point"},
				},
			},
			{
				input: `1.0.0`,
				output: Tokens{
					{Type: token.Error, Start: 0, End: 3, Value: "unexpected character after number: ."},
				},
			},
			{
				input: `1_0_0`,
				output: Tokens{
					{Type: token.ConstInt, Start: 0, End: 5, Value: "1_0_0"},
				},
			},
			{
				input:  `_1_0_0`,
				output: Tokens{},
			},
			{
				input: `1_0_0_`,
				output: Tokens{
					{Type: token.Error, Start: 0, End: 6, Value: "expected digit after each underscore"},
				},
			},
			{
				input: `0.1_0_0`,
				output: Tokens{
					{Type: token.ConstFloat, Start: 0, End: 7, Value: "0.1_0_0"},
				},
			},
			{
				input: `0.1__0_0`,
				output: Tokens{
					{Type: token.Error, Start: 0, End: 8, Value: "expected digit after each underscore"},
				},
			},
			{
				input:  `hello`,
				output: Tokens{},
			},
			{
				input: `1_200kb`,
				output: Tokens{
					{Type: token.ConstBytes, Start: 0, End: 7, Value: "1_200kb"},
				},
			},
		},
	)
}
