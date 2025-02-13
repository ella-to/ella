package main

import (
	"fmt"
	"os"
	"sort"
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
	cases, expected := getAllTestData(t, "./testdata/docs")
	counts := len(cases)

	for i := range counts {
		c := readContent(t, cases[i])
		e := readContent(t, expected[i])

		var sb strings.Builder
		parser := NewParser(c)

		result, err := ParseDocument(parser)
		if !assert.NoError(t, err) {
			return
		}

		result.Format(&sb)
		assert.Equal(t, e, sb.String())
	}
}

func readContent(t *testing.T, path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(content)
}

func getAllTestData(t *testing.T, path string) (cases []string, expected []string) {
	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.Contains(entry.Name(), "expected") {
			expected = append(expected, fmt.Sprintf("%s/%s", path, entry.Name()))
		} else {
			cases = append(cases, fmt.Sprintf("%s/%s", path, entry.Name()))
		}
	}

	if len(cases) != len(expected) {
		t.Fatalf("number of cases and expected files does not match")
	}

	sort.Strings(cases)
	sort.Strings(expected)

	return
}
