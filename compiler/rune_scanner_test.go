package compiler_test

import (
	"strings"
	"testing"

	"ella.to/ella/compiler"
)

func TestRuneScannerBuffer(t *testing.T) {
	for _, testCase := range []struct {
		input  string
		buffer string
		action func(rc *compiler.RuneScanner)
	}{
		{
			input:  "3.14",
			buffer: "3.14",
			action: func(rs *compiler.RuneScanner) {
				rs.AcceptRun("0123456789.")
			},
		},
	} {
		rs := compiler.NewRuneScanner(strings.NewReader(testCase.input), "test.ella")

		testCase.action(rs)

		gotBuffer := string(rs.Buffer())
		if gotBuffer != testCase.buffer {
			t.Errorf("for input %q, expected buffer %q, got %q", testCase.input, testCase.buffer, gotBuffer)
		}
	}
}
