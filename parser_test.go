package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParserValue(t *testing.T) {
	testCases := []struct {
		input  string
		output string
	}{
		{
			input:  `true`,
			output: `true`,
		},
		{
			input:  `false`,
			output: `false`,
		},
		{
			input:  `"hello"`,
			output: `"hello"`,
		},
		{
			input:  `123`,
			output: `123`,
		},
		{
			input:  `123.456`,
			output: `123.456`,
		},
		{
			input:  `null`,
			output: `null`,
		},
		{
			input:  `NewId`,
			output: `NewId`,
		},
		{
			input:  `1ns`,
			output: `1ns`,
		},
		{
			input:  `1us`,
			output: `1us`,
		},
		{
			input:  `1ms`,
			output: `1ms`,
		},
		{
			input:  `1s`,
			output: `1s`,
		},
		{
			input:  `1m`,
			output: `1m`,
		},
		{
			input:  `1h`,
			output: `1h`,
		},
		{
			input:  `1b`,
			output: `1b`,
		},
		{
			input:  `1kb`,
			output: `1kb`,
		},
		{
			input:  `1mb`,
			output: `1mb`,
		},
		{
			input:  `1gb`,
			output: `1gb`,
		},
		{
			input:  `1tb`,
			output: `1tb`,
		},
		{
			input:  `1pb`,
			output: `1pb`,
		},
		{
			input:  `1eb`,
			output: `1eb`,
		},
	}

	for _, tc := range testCases {
		var sb strings.Builder
		parser := NewParser(tc.input)

		result, err := ParseValue(parser)
		if !assert.NoError(t, err) {
			return
		}

		result.Format(&sb)
		assert.Equal(t, tc.output, sb.String())
	}
}

func TestParserConst(t *testing.T) {
	testCases := []struct {
		input  string
		output string
	}{
		{
			input:  `const A = true`,
			output: `const A = true`,
		},
		{
			input:  `const B = false`,
			output: `const B = false`,
		},
		{
			input:  `const C = "hello"`,
			output: `const C = "hello"`,
		},
		{
			input:  `const D = 123`,
			output: `const D = 123`,
		},
		{
			input:  `const E = 123.456`,
			output: `const E = 123.456`,
		},
		{
			input:  `const F = 123.456e-78`,
			output: `const F = 123.456e-78`,
		},
		{
			input:  `const G = 123.456e+78`,
			output: `const G = 123.456e+78`,
		},
		{
			input:  `const H = null`,
			output: `const H = null`,
		},
		{
			input:  `const I = NewId`,
			output: `const I = NewId`,
		},
		{
			input:  `const J = 1ns`,
			output: `const J = 1ns`,
		},
		{
			input:  `const K = 1us`,
			output: `const K = 1us`,
		},
		{
			input:  `const L = 1ms`,
			output: `const L = 1ms`,
		},
		{
			input:  `const M = 1s`,
			output: `const M = 1s`,
		},
		{
			input:  `const N = 1m`,
			output: `const N = 1m`,
		},
		{
			input:  `const O = 1h`,
			output: `const O = 1h`,
		},
		{
			input:  `const P = 1b`,
			output: `const P = 1b`,
		},
		{
			input:  `const Q = 1kb`,
			output: `const Q = 1kb`,
		},
		{
			input:  `const R = 1mb`,
			output: `const R = 1mb`,
		},
		{
			input:  `const S = 1gb`,
			output: `const S = 1gb`,
		},
		{
			input:  `const T = 1tb`,
			output: `const T = 1tb`,
		},
		{
			input:  `const U = 1pb`,
			output: `const U = 1pb`,
		},
		{
			input:  `const V = 1eb`,
			output: `const V = 1eb`,
		},
	}

	for _, tc := range testCases {
		var sb strings.Builder
		parser := NewParser(tc.input)

		result, err := ParseConst(parser)
		if !assert.NoError(t, err) {
			return
		}

		result.Format(&sb)
		assert.Equal(t, tc.output, sb.String())
	}
}

func TestParserDocument(t *testing.T) {
	testCases := []struct {
		input  string
		output string
	}{
		{
			input: `
model User {
	Id: string
	Name?: string
}
			`,
			output: `
model User {
    Id: string
    Name?: string
}`,
		},
	}

	for _, tc := range testCases {

		var sb strings.Builder
		parser := NewParser(tc.input)

		result, err := ParseDocument(parser)
		if !assert.NoError(t, err) {
			return
		}

		result.Format(&sb)
		assert.Equal(t, strings.TrimSpace(tc.output), sb.String())
	}
}

func TestParsComplex(t *testing.T) {
	testCases := []struct {
		input  string
		output string
		error  string
	}{
		{
			input: `
service RpcUserService {
    GetUserById(id: string) => (user: User)
}
			`,
			output: `
service RpcUserService {
    GetUserById (id: string) => (user: User)
}`,
		},
		{
			input: `
service HttpUserService {
    UploadAvatar(id: string, data: stream []byte)
}
					`,
			output: `
service HttpUserService {
    UploadAvatar (id: string, data: stream []byte)
}`,
		},
	}

	for _, tc := range testCases {
		var sb strings.Builder
		parser := NewParser(tc.input)

		result, err := ParseDocument(parser)
		if !assert.NoError(t, err) {
			return
		}

		result.Format(&sb)
		assert.Equal(t, strings.TrimSpace(tc.output), sb.String())
	}
}
