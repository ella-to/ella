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
			input: `model User {
				id: int64
				name?: string
			}`,
			output: Tokens{
				{Type: TokModel, Start: 0, End: 5, Value: "model"},
				{Type: TokIdentifier, Start: 6, End: 10, Value: "User"},
				{Type: TokOpenCurly, Start: 11, End: 12, Value: "{"},
				{Type: TokIdentifier, Start: 17, End: 19, Value: "id"},
				{Type: TokColon, Start: 19, End: 20, Value: ":"},
				{Type: TokInt64, Start: 21, End: 26, Value: "int64"},
				{Type: TokIdentifier, Start: 31, End: 35, Value: "name"},
				{Type: TokOptional, Start: 35, End: 36, Value: "?"},
				{Type: TokColon, Start: 36, End: 37, Value: ":"},
				{Type: TokString, Start: 38, End: 44, Value: "string"},
				{Type: TokCloseCurly, Start: 48, End: 49, Value: "}"},
				{Type: TokEOF, Start: 49, End: 49, Value: ""},
			},
		},
		{
			input: `service HttpFoo {
				GetAssetFile(assetId: string) => (result: stream []byte)
			}`,
			output: Tokens{
				{Type: TokService, Start: 0, End: 7, Value: "service"},
				{Type: TokIdentifier, Start: 8, End: 15, Value: "HttpFoo"},
				{Type: TokOpenCurly, Start: 16, End: 17, Value: "{"},
				{Type: TokIdentifier, Start: 22, End: 34, Value: "GetAssetFile"},
				{Type: TokOpenParen, Start: 34, End: 35, Value: "("},
				{Type: TokIdentifier, Start: 35, End: 42, Value: "assetId"},
				{Type: TokColon, Start: 42, End: 43, Value: ":"},
				{Type: TokString, Start: 44, End: 50, Value: "string"},
				{Type: TokCloseParen, Start: 50, End: 51, Value: ")"},
				{Type: TokReturn, Start: 52, End: 54, Value: "=>"},
				{Type: TokOpenParen, Start: 55, End: 56, Value: "("},
				{Type: TokIdentifier, Start: 56, End: 62, Value: "result"},
				{Type: TokColon, Start: 62, End: 63, Value: ":"},
				{Type: TokStream, Start: 64, End: 70, Value: "stream"},
				{Type: TokArray, Start: 71, End: 73, Value: "[]"},
				{Type: TokByte, Start: 73, End: 77, Value: "byte"},
				{Type: TokCloseParen, Start: 77, End: 78, Value: ")"},
				{Type: TokCloseCurly, Start: 82, End: 83, Value: "}"},
				{Type: TokEOF, Start: 83, End: 83, Value: ""},
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
				{Type: TokService, Start: 0, End: 7, Value: "service"},
				{Type: TokIdentifier, Start: 8, End: 14, Value: "RpcFoo"},
				{Type: TokOpenCurly, Start: 15, End: 16, Value: "{"},
				{Type: TokIdentifier, Start: 21, End: 27, Value: "GetFoo"},
				{Type: TokOpenParen, Start: 27, End: 28, Value: "("},
				{Type: TokCloseParen, Start: 28, End: 29, Value: ")"},
				{Type: TokReturn, Start: 30, End: 32, Value: "=>"},
				{Type: TokOpenParen, Start: 33, End: 34, Value: "("},
				{Type: TokIdentifier, Start: 34, End: 39, Value: "value"},
				{Type: TokColon, Start: 39, End: 40, Value: ":"},
				{Type: TokInt64, Start: 41, End: 46, Value: "int64"},
				{Type: TokCloseParen, Start: 46, End: 47, Value: ")"},
				{Type: TokOpenCurly, Start: 48, End: 49, Value: "{"},
				{Type: TokIdentifier, Start: 55, End: 63, Value: "Required"},
				{Type: TokIdentifier, Start: 69, End: 70, Value: "A"},
				{Type: TokAssign, Start: 71, End: 72, Value: "="},
				{Type: TokConstBytes, Start: 73, End: 76, Value: "1mb"},
				{Type: TokIdentifier, Start: 82, End: 83, Value: "B"},
				{Type: TokAssign, Start: 84, End: 85, Value: "="},
				{Type: TokConstDuration, Start: 86, End: 90, Value: "100h"},
				{Type: TokCloseCurly, Start: 95, End: 96, Value: "}"},
				{Type: TokCloseCurly, Start: 100, End: 101, Value: "}"},
				{Type: TokEOF, Start: 101, End: 101, Value: ""},
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

			service HttpMyService {
				GetUserById (id: int64) => (user: User) {
					method = "GET"
				}
			}

			`,
			output: Tokens{
				{Type: TokIdentifier, Start: 5, End: 6, Value: "a"},
				{Type: TokAssign, Start: 7, End: 8, Value: "="},
				{Type: TokConstFloat, Start: 9, End: 12, Value: "1.0"},
				{Type: TokIdentifier, Start: 17, End: 24, Value: "message"},
				{Type: TokIdentifier, Start: 25, End: 26, Value: "A"},
				{Type: TokOpenCurly, Start: 27, End: 28, Value: "{"},
				{Type: TokIdentifier, Start: 33, End: 42, Value: "firstname"},
				{Type: TokColon, Start: 42, End: 43, Value: ":"},
				{Type: TokString, Start: 44, End: 50, Value: "string"},
				{Type: TokOpenCurly, Start: 51, End: 52, Value: "{"},
				{Type: TokIdentifier, Start: 58, End: 66, Value: "required"},
				{Type: TokIdentifier, Start: 72, End: 79, Value: "pattern"},
				{Type: TokAssign, Start: 80, End: 81, Value: "="},
				{Type: TokConstStringDoubleQuote, Start: 83, End: 94, Value: "^[a-zA-Z]+$"},
				{Type: TokCloseCurly, Start: 100, End: 101, Value: "}"},
				{Type: TokCloseCurly, Start: 105, End: 106, Value: "}"},
				{Type: TokService, Start: 111, End: 118, Value: "service"},
				{Type: TokIdentifier, Start: 119, End: 132, Value: "HttpMyService"},
				{Type: TokOpenCurly, Start: 133, End: 134, Value: "{"},
				{Type: TokIdentifier, Start: 139, End: 150, Value: "GetUserById"},
				{Type: TokOpenParen, Start: 151, End: 152, Value: "("},
				{Type: TokIdentifier, Start: 152, End: 154, Value: "id"},
				{Type: TokColon, Start: 154, End: 155, Value: ":"},
				{Type: TokInt64, Start: 156, End: 161, Value: "int64"},
				{Type: TokCloseParen, Start: 161, End: 162, Value: ")"},
				{Type: TokReturn, Start: 163, End: 165, Value: "=>"},
				{Type: TokOpenParen, Start: 166, End: 167, Value: "("},
				{Type: TokIdentifier, Start: 167, End: 171, Value: "user"},
				{Type: TokColon, Start: 171, End: 172, Value: ":"},
				{Type: TokIdentifier, Start: 173, End: 177, Value: "User"},
				{Type: TokCloseParen, Start: 177, End: 178, Value: ")"},
				{Type: TokOpenCurly, Start: 179, End: 180, Value: "{"},
				{Type: TokIdentifier, Start: 186, End: 192, Value: "method"},
				{Type: TokAssign, Start: 193, End: 194, Value: "="},
				{Type: TokConstStringDoubleQuote, Start: 196, End: 199, Value: "GET"},
				{Type: TokCloseCurly, Start: 205, End: 206, Value: "}"},
				{Type: TokCloseCurly, Start: 210, End: 211, Value: "}"},
				{Type: TokEOF, Start: 216, End: 216, Value: ""},
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
