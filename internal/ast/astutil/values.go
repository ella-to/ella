package astutil

import (
	"strings"

	"ella.to/ella/internal/ast"
	"ella.to/ella/pkg/strcase"
)

type MethodOptions struct {
	HttpMethod    string
	ContentType   string // only used for Download methods or stream []byte
	MaxUploadSize int64
	RawControl    bool
}

func ParseMethodOptions(options ast.Options) MethodOptions {
	mapper := make(map[string]any)
	for _, opt := range options {
		var value any
		switch v := opt.Value.(type) {
		case *ast.ValueString:
			value = v.Value
		case *ast.ValueBool:
			value = v.Value
		case *ast.ValueInt:
			value = v.Value
		case *ast.ValueFloat:
			value = v.Value
		case *ast.ValueByteSize:
			value = v.Value * int64(v.Scale)
		case *ast.ValueDuration:
			value = v.Value * int64(v.Scale)
		default:
			value = opt.Value
		}

		mapper[strcase.ToPascal(opt.Name.Token.Literal)] = value
	}

	return MethodOptions{
		HttpMethod:    strings.ToUpper(castString(mapper["HttpMethod"], "POST")),
		ContentType:   castString(mapper["ContentType"], "application/octet-stream"),
		MaxUploadSize: castInt64(mapper["MaxUploadSize"], 1*1024*1024),
		RawControl:    castBool(mapper["RawControl"], false),
	}
}

func castString(value any, defaultValue string) string {
	return castValue[string](value, defaultValue)
}

func castInt64(value any, defaultValue int64) int64 {
	return castValue[int64](value, defaultValue)
}

func castBool(value any, defaultValue bool) bool {
	return castValue[bool](value, defaultValue)
}

func castValue[T any](value any, defaultValue T) T {
	result, ok := value.(T)
	if ok {
		return result
	}

	return defaultValue
}
