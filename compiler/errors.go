package compiler

import (
	"fmt"
	"strings"
)

// Error represents a unified error type for scanner, parser, and validation errors
type Error struct {
	Token  *Token
	Reason string
}

func (e *Error) Error() string {
	if e.Token != nil {
		return fmt.Sprintf("error at line %d, column %d: %s", e.Token.Pos.Line, e.Token.Pos.Column, e.Reason)
	}
	return fmt.Sprintf("error: %s", e.Reason)
}

// NewError creates a new Error with the given token and reason
func NewError(tok *Token, format string, args ...any) *Error {
	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}
	return &Error{
		Token:  tok,
		Reason: format,
	}
}

// ErrorDisplay provides formatted error output with source context
type ErrorDisplay struct {
	source   string
	lines    []string
	filename string
}

// NewErrorDisplay creates a new ErrorDisplay from source code
func NewErrorDisplay(source string, filename string) *ErrorDisplay {
	lines := strings.Split(source, "\n")
	return &ErrorDisplay{
		source:   source,
		lines:    lines,
		filename: filename,
	}
}

// FormatError formats an Error with source context for terminal display
func (ed *ErrorDisplay) FormatError(err error) string {
	compilerErr, ok := err.(*Error)
	if !ok {
		return err.Error()
	}

	return ed.FormatCompilerError(compilerErr)
}

// FormatCompilerError formats an Error with source context
func (ed *ErrorDisplay) FormatCompilerError(err *Error) string {
	var sb strings.Builder

	line := err.Token.Pos.Line
	col := err.Token.Pos.Column

	// Header with error location
	if ed.filename != "" {
		sb.WriteString(fmt.Sprintf("\n\033[1;31merror\033[0m: %s\n", err.Reason))
		sb.WriteString(fmt.Sprintf("  \033[1;36m-->\033[0m %s:%d:%d\n", ed.filename, line, col))
	} else {
		sb.WriteString(fmt.Sprintf("\n\033[1;31merror\033[0m: %s\n", err.Reason))
		sb.WriteString(fmt.Sprintf("  \033[1;36m-->\033[0m line %d, column %d\n", line, col))
	}

	sb.WriteString("   \033[1;36m|\033[0m\n")

	// Calculate context range (2-3 lines before and after)
	contextBefore := 3
	contextAfter := 2
	startLine := max(line-contextBefore, 1)
	endLine := min(line+contextAfter, len(ed.lines))

	// Calculate the width needed for line numbers
	lineNumWidth := len(fmt.Sprintf("%d", endLine))

	// Print context lines before the error
	for i := startLine; i < line; i++ {
		if i > 0 && i <= len(ed.lines) {
			sb.WriteString(fmt.Sprintf("\033[1;36m%*d |\033[0m %s\n", lineNumWidth, i, ed.lines[i-1]))
		}
	}

	// Print the error line with highlighting
	if line > 0 && line <= len(ed.lines) {
		errorLine := ed.lines[line-1]
		sb.WriteString(fmt.Sprintf("\033[1;36m%*d |\033[0m %s\n", lineNumWidth, line, errorLine))

		// Print the error pointer
		// We need to preserve tabs from the original line for correct alignment
		padCol := max(col-1, 0)
		var padding strings.Builder
		for i := 0; i < padCol && i < len(errorLine); i++ {
			if errorLine[i] == '\t' {
				padding.WriteByte('\t')
			} else {
				padding.WriteByte(' ')
			}
		}
		// If col is beyond the line length, pad with spaces
		for i := len(errorLine); i < padCol; i++ {
			padding.WriteByte(' ')
		}
		tokenLen := max(len(err.Token.Lit), 1)
		pointer := strings.Repeat("^", tokenLen)

		sb.WriteString(fmt.Sprintf("\033[1;36m%*s |\033[0m %s\033[1;31m%s\033[0m \033[1;31m%s\033[0m\n",
			lineNumWidth, "", padding.String(), pointer, err.Reason))
	}

	// Print context lines after the error
	for i := line + 1; i <= endLine; i++ {
		if i > 0 && i <= len(ed.lines) {
			sb.WriteString(fmt.Sprintf("\033[1;36m%*d |\033[0m %s\n", lineNumWidth, i, ed.lines[i-1]))
		}
	}

	sb.WriteString("   \033[1;36m|\033[0m\n")

	return sb.String()
}

// FormatErrorPlain formats an error without ANSI colors (for non-terminal output)
func (ed *ErrorDisplay) FormatErrorPlain(err error) string {
	compilerErr, ok := err.(*Error)
	if !ok {
		return err.Error()
	}

	return ed.FormatCompilerErrorPlain(compilerErr)
}

// FormatCompilerErrorPlain formats an Error without ANSI colors
func (ed *ErrorDisplay) FormatCompilerErrorPlain(err *Error) string {
	var sb strings.Builder

	line := err.Token.Pos.Line
	col := err.Token.Pos.Column

	// Header with error location
	if ed.filename != "" {
		sb.WriteString(fmt.Sprintf("\nerror: %s\n", err.Reason))
		sb.WriteString(fmt.Sprintf("  --> %s:%d:%d\n", ed.filename, line, col))
	} else {
		sb.WriteString(fmt.Sprintf("\nerror: %s\n", err.Reason))
		sb.WriteString(fmt.Sprintf("  --> line %d, column %d\n", line, col))
	}

	sb.WriteString("   |\n")

	// Calculate context range
	contextBefore := 3
	contextAfter := 2
	startLine := line - contextBefore
	if startLine < 1 {
		startLine = 1
	}
	endLine := line + contextAfter
	if endLine > len(ed.lines) {
		endLine = len(ed.lines)
	}

	// Calculate the width needed for line numbers
	lineNumWidth := len(fmt.Sprintf("%d", endLine))

	// Print context lines before the error
	for i := startLine; i < line; i++ {
		if i > 0 && i <= len(ed.lines) {
			sb.WriteString(fmt.Sprintf("%*d | %s\n", lineNumWidth, i, ed.lines[i-1]))
		}
	}

	// Print the error line
	if line > 0 && line <= len(ed.lines) {
		errorLine := ed.lines[line-1]
		sb.WriteString(fmt.Sprintf("%*d | %s\n", lineNumWidth, line, errorLine))

		// Print the error pointer
		// We need to preserve tabs from the original line for correct alignment
		padCol := col - 1
		if padCol < 0 {
			padCol = 0
		}
		var padding strings.Builder
		for i := 0; i < padCol && i < len(errorLine); i++ {
			if errorLine[i] == '\t' {
				padding.WriteByte('\t')
			} else {
				padding.WriteByte(' ')
			}
		}
		// If col is beyond the line length, pad with spaces
		for i := len(errorLine); i < padCol; i++ {
			padding.WriteByte(' ')
		}
		tokenLen := len(err.Token.Lit)
		if tokenLen < 1 {
			tokenLen = 1
		}
		pointer := strings.Repeat("^", tokenLen)

		sb.WriteString(fmt.Sprintf("%*s | %s%s %s\n", lineNumWidth, "", padding.String(), pointer, err.Reason))
	}

	// Print context lines after the error
	for i := line + 1; i <= endLine; i++ {
		if i > 0 && i <= len(ed.lines) {
			sb.WriteString(fmt.Sprintf("%*d | %s\n", lineNumWidth, i, ed.lines[i-1]))
		}
	}

	sb.WriteString("   |\n")

	return sb.String()
}
