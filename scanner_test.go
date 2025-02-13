package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Tokens []Token

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
		sb.WriteString(fmt.Sprintf("{Type: Tok%s, Start: %d, End: %d, Value: \"%s\"},\n", t[i].Type, t[i].Start, t[i].End, t[i].Value))
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
		emitter := TokenEmitterFunc(func(token *Token) {
			output = append(output, *token)
		})

		Start(emitter, initState, tc.input)
		assert.Equal(t, tc.output, output, "Failed scanner at %d: %s", i, output)
	}
}

func TestLex(t *testing.T) {
	runTestCase(t, -1, Lex, TestCases{
		{
			input: `service Foo {
				http GetAssetFile(assetId: string) => (result: file)
			}`,
			output: Tokens{
				{Type: TokService, Start: 0, End: 7, Value: "service"},
				{Type: TokIdentifier, Start: 8, End: 11, Value: "Foo"},
				{Type: TokOpenCurly, Start: 12, End: 13, Value: "{"},
				{Type: TokHttp, Start: 18, End: 22, Value: "http"},
				{Type: TokIdentifier, Start: 23, End: 35, Value: "GetAssetFile"},
				{Type: TokOpenParen, Start: 35, End: 36, Value: "("},
				{Type: TokIdentifier, Start: 36, End: 43, Value: "assetId"},
				{Type: TokColon, Start: 43, End: 44, Value: ":"},
				{Type: TokString, Start: 45, End: 51, Value: "string"},
				{Type: TokCloseParen, Start: 51, End: 52, Value: ")"},
				{Type: TokReturn, Start: 53, End: 55, Value: "=>"},
				{Type: TokOpenParen, Start: 56, End: 57, Value: "("},
				{Type: TokIdentifier, Start: 57, End: 63, Value: "result"},
				{Type: TokColon, Start: 63, End: 64, Value: ":"},
				{Type: TokFile, Start: 65, End: 69, Value: "file"},
				{Type: TokCloseParen, Start: 69, End: 70, Value: ")"},
				{Type: TokCloseCurly, Start: 74, End: 75, Value: "}"},
				{Type: TokEOF, Start: 75, End: 75, Value: ""},
			},
		},
		{
			input: `service Foo {
				rpc GetFoo() => (value: int64) {
					Required
					A = 1mb
					B = 100h
				}
			}`,
			output: Tokens{
				{Type: TokService, Start: 0, End: 7, Value: "service"},
				{Type: TokIdentifier, Start: 8, End: 11, Value: "Foo"},
				{Type: TokOpenCurly, Start: 12, End: 13, Value: "{"},
				{Type: TokRpc, Start: 18, End: 21, Value: "rpc"},
				{Type: TokIdentifier, Start: 22, End: 28, Value: "GetFoo"},
				{Type: TokOpenParen, Start: 28, End: 29, Value: "("},
				{Type: TokCloseParen, Start: 29, End: 30, Value: ")"},
				{Type: TokReturn, Start: 31, End: 33, Value: "=>"},
				{Type: TokOpenParen, Start: 34, End: 35, Value: "("},
				{Type: TokIdentifier, Start: 35, End: 40, Value: "value"},
				{Type: TokColon, Start: 40, End: 41, Value: ":"},
				{Type: TokInt64, Start: 42, End: 47, Value: "int64"},
				{Type: TokCloseParen, Start: 47, End: 48, Value: ")"},
				{Type: TokOpenCurly, Start: 49, End: 50, Value: "{"},
				{Type: TokIdentifier, Start: 56, End: 64, Value: "Required"},
				{Type: TokIdentifier, Start: 70, End: 71, Value: "A"},
				{Type: TokAssign, Start: 72, End: 73, Value: "="},
				{Type: TokConstBytes, Start: 74, End: 77, Value: "1mb"},
				{Type: TokIdentifier, Start: 83, End: 84, Value: "B"},
				{Type: TokAssign, Start: 85, End: 86, Value: "="},
				{Type: TokConstDuration, Start: 87, End: 91, Value: "100h"},
				{Type: TokCloseCurly, Start: 96, End: 97, Value: "}"},
				{Type: TokCloseCurly, Start: 101, End: 102, Value: "}"},
				{Type: TokEOF, Start: 102, End: 102, Value: ""},
			},
		},
		{
			input: `A = 1mb`,
			output: Tokens{
				{Type: TokIdentifier, Start: 0, End: 1, Value: "A"},
				{Type: TokAssign, Start: 2, End: 3, Value: "="},
				{Type: TokConstBytes, Start: 4, End: 7, Value: "1mb"},
				{Type: TokEOF, Start: 7, End: 7, Value: ""},
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
				{Type: TokComment, Start: 9, End: 29, Value: " this is a comment 1"},
				{Type: TokComment, Start: 34, End: 60, Value: " this is another comment 2"},
				{Type: TokIdentifier, Start: 64, End: 65, Value: "a"},
				{Type: TokAssign, Start: 66, End: 67, Value: "="},
				{Type: TokConstInt, Start: 68, End: 69, Value: "1"},
				{Type: TokComment, Start: 71, End: 91, Value: " this is a comment 3"},
				{Type: TokComment, Start: 96, End: 122, Value: " this is another comment 4"},
				{Type: TokIdentifier, Start: 127, End: 134, Value: "message"},
				{Type: TokIdentifier, Start: 135, End: 136, Value: "A"},
				{Type: TokOpenCurly, Start: 137, End: 138, Value: "{"},
				{Type: TokComment, Start: 144, End: 164, Value: " this is a comment 5"},
				{Type: TokComment, Start: 170, End: 196, Value: " this is another comment 6"},
				{Type: TokIdentifier, Start: 201, End: 210, Value: "firstname"},
				{Type: TokColon, Start: 210, End: 211, Value: ":"},
				{Type: TokString, Start: 212, End: 218, Value: "string"},
				{Type: TokCloseCurly, Start: 222, End: 223, Value: "}"},
				{Type: TokEOF, Start: 231, End: 231, Value: ""},
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
				{Type: TokComment, Start: 6, End: 30, Value: " This is a first comment"},
				{Type: TokIdentifier, Start: 34, End: 35, Value: "a"},
				{Type: TokAssign, Start: 36, End: 37, Value: "="},
				{Type: TokConstInt, Start: 38, End: 39, Value: "1"},
				{Type: TokComment, Start: 41, End: 68, Value: " this is the second comment"},
				{Type: TokComment, Start: 73, End: 99, Value: " this is the third comment"},
				{Type: TokEOF, Start: 105, End: 105, Value: ""},
			},
		},
		{
			input: `ella = "1.0.0-b01"`,
			output: Tokens{
				{Type: TokIdentifier, Start: 0, End: 4, Value: "ella"},
				{Type: TokAssign, Start: 5, End: 6, Value: "="},
				{Type: TokConstStringDoubleQuote, Start: 8, End: 17, Value: "1.0.0-b01"},
				{Type: TokEOF, Start: 18, End: 18, Value: ""},
			},
		},
		{
			input: `message A {
				...B
				...C

				first: int64
			}`,
			output: Tokens{
				{Type: TokIdentifier, Start: 0, End: 7, Value: "message"},
				{Type: TokIdentifier, Start: 8, End: 9, Value: "A"},
				{Type: TokOpenCurly, Start: 10, End: 11, Value: "{"},
				{Type: TokExtend, Start: 16, End: 19, Value: "..."},
				{Type: TokIdentifier, Start: 19, End: 20, Value: "B"},
				{Type: TokExtend, Start: 25, End: 28, Value: "..."},
				{Type: TokIdentifier, Start: 28, End: 29, Value: "C"},
				{Type: TokIdentifier, Start: 35, End: 40, Value: "first"},
				{Type: TokColon, Start: 40, End: 41, Value: ":"},
				{Type: TokInt64, Start: 42, End: 47, Value: "int64"},
				{Type: TokCloseCurly, Start: 51, End: 52, Value: "}"},
				{Type: TokEOF, Start: 52, End: 52, Value: ""},
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
				{Type: TokEnum, Start: 0, End: 4, Value: "enum"},
				{Type: TokIdentifier, Start: 5, End: 6, Value: "a"},
				{Type: TokInt64, Start: 7, End: 12, Value: "int64"},
				{Type: TokOpenCurly, Start: 13, End: 14, Value: "{"},
				{Type: TokIdentifier, Start: 19, End: 22, Value: "one"},
				{Type: TokAssign, Start: 23, End: 24, Value: "="},
				{Type: TokConstInt, Start: 25, End: 26, Value: "1"},
				{Type: TokComment, Start: 28, End: 36, Value: " comment"},
				{Type: TokIdentifier, Start: 41, End: 44, Value: "two"},
				{Type: TokAssign, Start: 45, End: 46, Value: "="},
				{Type: TokConstInt, Start: 47, End: 48, Value: "2"},
				{Type: TokComment, Start: 49, End: 58, Value: " comment2"},
				{Type: TokIdentifier, Start: 63, End: 68, Value: "three"},
				{Type: TokCloseCurly, Start: 72, End: 73, Value: "}"},
				{Type: TokEOF, Start: 73, End: 73, Value: ""},
			},
		},
		{
			input: `enum a int64 {
				one = 1
				two = 2
				three
			}`,
			output: Tokens{
				{Type: TokEnum, Start: 0, End: 4, Value: "enum"},
				{Type: TokIdentifier, Start: 5, End: 6, Value: "a"},
				{Type: TokInt64, Start: 7, End: 12, Value: "int64"},
				{Type: TokOpenCurly, Start: 13, End: 14, Value: "{"},
				{Type: TokIdentifier, Start: 19, End: 22, Value: "one"},
				{Type: TokAssign, Start: 23, End: 24, Value: "="},
				{Type: TokConstInt, Start: 25, End: 26, Value: "1"},
				{Type: TokIdentifier, Start: 31, End: 34, Value: "two"},
				{Type: TokAssign, Start: 35, End: 36, Value: "="},
				{Type: TokConstInt, Start: 37, End: 38, Value: "2"},
				{Type: TokIdentifier, Start: 43, End: 48, Value: "three"},
				{Type: TokCloseCurly, Start: 52, End: 53, Value: "}"},
				{Type: TokEOF, Start: 53, End: 53, Value: ""},
			},
		},
		{
			input: `enum a int64 {}`,
			output: Tokens{
				{Type: TokEnum, Start: 0, End: 4, Value: "enum"},
				{Type: TokIdentifier, Start: 5, End: 6, Value: "a"},
				{Type: TokInt64, Start: 7, End: 12, Value: "int64"},
				{Type: TokOpenCurly, Start: 13, End: 14, Value: "{"},
				{Type: TokCloseCurly, Start: 14, End: 15, Value: "}"},
				{Type: TokEOF, Start: 15, End: 15, Value: ""},
			},
		},
		{
			input: `a=1`,
			output: Tokens{
				{Type: TokIdentifier, Start: 0, End: 1, Value: "a"},
				{Type: TokAssign, Start: 1, End: 2, Value: "="},
				{Type: TokConstInt, Start: 2, End: 3, Value: "1"},
				{Type: TokEOF, Start: 3, End: 3, Value: ""},
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

			service MyService {
				http GetUserById (id: int64) => (user: User) {
					method = "GET"
				}
			}
			
			`,
			output: Tokens{
				{Type: TokIdentifier, Start: 8, End: 9, Value: "a"},
				{Type: TokAssign, Start: 10, End: 11, Value: "="},
				{Type: TokConstFloat, Start: 12, End: 15, Value: "1.0"},
				{Type: TokIdentifier, Start: 20, End: 27, Value: "message"},
				{Type: TokIdentifier, Start: 28, End: 29, Value: "A"},
				{Type: TokOpenCurly, Start: 30, End: 31, Value: "{"},
				{Type: TokIdentifier, Start: 36, End: 45, Value: "firstname"},
				{Type: TokColon, Start: 45, End: 46, Value: ":"},
				{Type: TokString, Start: 47, End: 53, Value: "string"},
				{Type: TokOpenCurly, Start: 54, End: 55, Value: "{"},
				{Type: TokIdentifier, Start: 61, End: 69, Value: "required"},
				{Type: TokIdentifier, Start: 75, End: 82, Value: "pattern"},
				{Type: TokAssign, Start: 83, End: 84, Value: "="},
				{Type: TokConstStringDoubleQuote, Start: 86, End: 97, Value: "^[a-zA-Z]+$"},
				{Type: TokCloseCurly, Start: 103, End: 104, Value: "}"},
				{Type: TokCloseCurly, Start: 108, End: 109, Value: "}"},
				{Type: TokService, Start: 114, End: 121, Value: "service"},
				{Type: TokIdentifier, Start: 122, End: 131, Value: "MyService"},
				{Type: TokOpenCurly, Start: 132, End: 133, Value: "{"},
				{Type: TokHttp, Start: 138, End: 142, Value: "http"},
				{Type: TokIdentifier, Start: 143, End: 154, Value: "GetUserById"},
				{Type: TokOpenParen, Start: 155, End: 156, Value: "("},
				{Type: TokIdentifier, Start: 156, End: 158, Value: "id"},
				{Type: TokColon, Start: 158, End: 159, Value: ":"},
				{Type: TokInt64, Start: 160, End: 165, Value: "int64"},
				{Type: TokCloseParen, Start: 165, End: 166, Value: ")"},
				{Type: TokReturn, Start: 167, End: 169, Value: "=>"},
				{Type: TokOpenParen, Start: 170, End: 171, Value: "("},
				{Type: TokIdentifier, Start: 171, End: 175, Value: "user"},
				{Type: TokColon, Start: 175, End: 176, Value: ":"},
				{Type: TokIdentifier, Start: 177, End: 181, Value: "User"},
				{Type: TokCloseParen, Start: 181, End: 182, Value: ")"},
				{Type: TokOpenCurly, Start: 183, End: 184, Value: "{"},
				{Type: TokIdentifier, Start: 190, End: 196, Value: "method"},
				{Type: TokAssign, Start: 197, End: 198, Value: "="},
				{Type: TokConstStringDoubleQuote, Start: 200, End: 203, Value: "GET"},
				{Type: TokCloseCurly, Start: 209, End: 210, Value: "}"},
				{Type: TokCloseCurly, Start: 214, End: 215, Value: "}"},
				{Type: TokEOF, Start: 223, End: 223, Value: ""},
			},
		},
		{
			input: `error ErrUserNotFound { Code = 1000 HttpStatus = NotFound Msg = "user not found" }`,
			output: Tokens{
				{Type: TokCustomError, Start: 0, End: 5, Value: "error"},
				{Type: TokIdentifier, Start: 6, End: 21, Value: "ErrUserNotFound"},
				{Type: TokOpenCurly, Start: 22, End: 23, Value: "{"},
				{Type: TokIdentifier, Start: 24, End: 28, Value: "Code"},
				{Type: TokAssign, Start: 29, End: 30, Value: "="},
				{Type: TokConstInt, Start: 31, End: 35, Value: "1000"},
				{Type: TokIdentifier, Start: 36, End: 46, Value: "HttpStatus"},
				{Type: TokAssign, Start: 47, End: 48, Value: "="},
				{Type: TokIdentifier, Start: 49, End: 57, Value: "NotFound"},
				{Type: TokIdentifier, Start: 58, End: 61, Value: "Msg"},
				{Type: TokAssign, Start: 62, End: 63, Value: "="},
				{Type: TokConstStringDoubleQuote, Start: 65, End: 79, Value: "user not found"},
				{Type: TokCloseCurly, Start: 81, End: 82, Value: "}"},
				{Type: TokEOF, Start: 82, End: 82, Value: ""},
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
					{Type: TokConstInt, Start: 0, End: 1, Value: "1"},
				},
			},
			{
				input: `1.0`,
				output: Tokens{
					{Type: TokConstFloat, Start: 0, End: 3, Value: "1.0"},
				},
			},
			{
				input: `1.`,
				output: Tokens{
					{Type: TokError, Start: 0, End: 2, Value: "expected digit after decimal point"},
				},
			},
			{
				input: `1.0.0`,
				output: Tokens{
					{Type: TokError, Start: 0, End: 3, Value: "unexpected character after number: ."},
				},
			},
			{
				input: `1_0_0`,
				output: Tokens{
					{Type: TokConstInt, Start: 0, End: 5, Value: "1_0_0"},
				},
			},
			{
				input:  `_1_0_0`,
				output: Tokens{},
			},
			{
				input: `1_0_0_`,
				output: Tokens{
					{Type: TokError, Start: 0, End: 6, Value: "expected digit after each underscore"},
				},
			},
			{
				input: `0.1_0_0`,
				output: Tokens{
					{Type: TokConstFloat, Start: 0, End: 7, Value: "0.1_0_0"},
				},
			},
			{
				input: `0.1__0_0`,
				output: Tokens{
					{Type: TokError, Start: 0, End: 8, Value: "expected digit after each underscore"},
				},
			},
			{
				input:  `hello`,
				output: Tokens{},
			},
			{
				input: `1_200kb`,
				output: Tokens{
					{Type: TokConstBytes, Start: 0, End: 7, Value: "1_200kb"},
				},
			},
		},
	)
}
