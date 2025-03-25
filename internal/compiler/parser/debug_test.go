package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrintMessage(t *testing.T) {
	const source = `
enum A {
}

model B {
	hello
}
`

	result := PrettyMessage("test.ella", source, 14, 15, "test error")

	expected := `Error: test error at (test.ella) line 5, column 2

   2 | enum A {
   3 | }
   4 | 
   5 | model B {
     |  ^
   6 | 	hello
   7 | }
   8 | 
`

	assert.Equal(t, expected, result)
}
