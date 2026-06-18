package compiler

import (
	"strings"
	"testing"
)

func TestWasmGenerator_ServiceOptionalArgs(t *testing.T) {
	source := `service UserService {
	List (query: string, limit?: int64, tags?: []string) => (ids: []string)
}
`
	scanner := NewScanner(strings.NewReader(withModule(source)), "test.ella")
	parser := NewParser(scanner)
	program, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gen := NewWasmGenerator(program, "main", false)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	// Call options shift one slot to the right to make room for the
	// optional-args object: [query, optionalArgs, options]
	if !strings.Contains(code, "opts := createJsCallOptions(args, 2)") {
		t.Errorf("expected call options at index 2, got:\n%s", code)
	}

	// Optional-args object is read from index 1
	if !strings.Contains(code, "_optionalJS := jsGetArg(args, 1)") {
		t.Errorf("expected optional args object at index 1, got:\n%s", code)
	}

	// Present keys become shared With options
	if !strings.Contains(code, "_opts = append(_opts, WithLimit(int64(_limitJS.Int())))") {
		t.Errorf("expected WithLimit conversion, got:\n%s", code)
	}
	if !strings.Contains(code, "_opts = append(_opts, WithTags(_tags))") {
		t.Errorf("expected WithTags conversion, got:\n%s", code)
	}

	// Options forwarded variadically to the client interface
	if !strings.Contains(code, "serviceImpl.List(ctx, query, _opts...)") {
		t.Errorf("expected variadic options in impl call, got:\n%s", code)
	}
}
