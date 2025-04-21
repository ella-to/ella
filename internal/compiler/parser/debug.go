package parser

import (
	"fmt"
	"os"
	"strings"

	"ella.to/ella/internal/compiler/token"
)

type Error struct {
	Filename string
	Start    int
	End      int
	Message  string
}

func (e *Error) Error() string {
	return PrettyMessageWithFilename(e.Filename, e.Start, e.End, e.Message)
}

func NewError(tok *token.Token, format string, args ...any) error {
	return &Error{
		Filename: tok.Filename,
		Start:    tok.Start,
		End:      tok.End,
		Message:  fmt.Sprintf(format, args...),
	}
}

func NewErrorWithEndToken(start *token.Token, end *token.Token, format string, args ...any) error {
	return &Error{
		Filename: start.Filename,
		Start:    start.Start,
		End:      end.End,
		Message:  fmt.Sprintf(format, args...),
	}
}

func PrettyMessageWithFilename(filename string, start int, end int, msg string) string {
	b, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Sprintf("Error: %s\n", msg)
	}

	return PrettyMessage(filename, string(b), start, end, msg)
}

func PrettyMessage(filename string, src string, start int, end int, msg string) string {
	lines := strings.Split(src, "\n")
	lineStart, column := getLineAndColumn(src, start)

	var output strings.Builder

	// Print error message with line and column
	if filename != "" {
		fmt.Fprintf(&output, "Error: %s at (%s:%d:%d)\n\n", msg, filename, lineStart+1, column+1)
	} else {
		fmt.Fprintf(&output, "Error: %s at line %d, column %d\n\n", msg, lineStart+1, column+1)
	}

	// Show context (3 lines before and after)
	startLine := max(0, lineStart-3)
	endLine := min(len(lines), lineStart+4)

	for i := startLine; i < endLine; i++ {
		// Line number
		fmt.Fprintf(&output, "%4d | %s\n", i+1, lines[i])

		// Print caret under the error
		if i == lineStart {
			fmt.Fprintf(&output, "     | %s%s\n",
				strings.Repeat(" ", column),
				strings.Repeat("^", end-start))
		}
	}

	return output.String()
}

func getLineAndColumn(source string, pos int) (line, col int) {
	line = strings.Count(source[:pos], "\n")
	if line == 0 {
		return 0, pos
	}
	lastNewline := strings.LastIndex(source[:pos], "\n")
	return line, pos - lastNewline - 1
}
