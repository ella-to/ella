package gen

import (
	"embed"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"

	"ella.to/ella/internal/compiler/ast"
	"ella.to/ella/internal/compiler/token"
	"ella.to/ella/internal/strcase"
)

//go:embed golang/*.go.tmpl
var golangTemplateFiles embed.FS

func generateGo(pkg, output string, doc *ast.Document) error {
	// CONSTANTS

	type MethodType int

	const (
		MethodJsonToJson     MethodType = iota // 0
		MethodJsonToSSE                        // 1
		MethodJsonToBinary                     // 2 ok
		MethodBinaryToJson                     // 3
		MethodBinaryToSSE                      // 4
		MethodBinaryToBinary                   // 5
	)

	type GoConst struct {
		Name  string
		Value string
	}

	// ENUMS

	type GoEnumKeyValue struct {
		Name  string
		Value string
	}

	type GoEnum struct {
		Name string
		Type string // int8, int16, int32, int64
		Keys []GoEnumKeyValue
	}

	// MODELS

	type GoModelField struct {
		Name string
		Type string
		Tags string
	}

	type GoModel struct {
		Name   string
		Fields []GoModelField
	}

	// SERVICES

	type GoMethodArg struct {
		Name   string
		Type   string
		Stream bool
	}

	type GoMethodReturn struct {
		Name   string
		Type   string
		Stream bool
	}

	type GoMethodOption struct {
		Name  string
		Value any
	}

	type GoMethod struct {
		Name        string
		ServiceName string // add this so it would be easier to generate the service path
		Args        []GoMethodArg
		Returns     []GoMethodReturn
		Options     []GoMethodOption

		Type         MethodType
		Timeout      int64
		TotalMaxSize int64
	}

	type GoService struct {
		Name    string
		Methods []GoMethod
	}

	// ERRORS

	type GoError struct {
		Name    string
		Code    int64
		Message string
	}

	type Data struct {
		PackageName   string
		Constants     []GoConst
		Enums         []GoEnum
		Models        []GoModel
		HttpServices  []GoService
		RpcServices   []GoService
		Errors        []GoError
		Json2Json     set[int] // set of method's returns size
		Json2Binary   bool
		Json2SSE      bool
		Binary2Json   set[int] // set of method's returns size
		Binary2Binary bool
		Binary2SSE    bool
	}

	tmpl, err := template.
		New("GenerateGo").
		Funcs(defaultFuncsMap).
		Funcs(template.FuncMap{
			// Generate a list of generics for the method arguments and returns
			// e.g. A, R1, R2, R3 any
			"GenArgsGenerics": func(size int) string {
				var sb strings.Builder

				sb.WriteString("A")
				for i := 1; i <= size; i++ {
					sb.WriteString(fmt.Sprintf(", R%d", i))
				}
				sb.WriteString(" any")

				return sb.String()
			},
			// Generate a list of generics for the method returns
			// e.g. R1, R2, R3, error
			"GenReturnsGenerics": func(size int) string {
				var sb strings.Builder

				for i := 1; i <= size; i++ {
					if i > 1 {
						sb.WriteString(", ")
					}
					sb.WriteString(fmt.Sprintf("R%d", i))
				}

				if size > 0 {
					sb.WriteString(", ")
				}

				sb.WriteString("error")

				return sb.String()
			},
			"GetHandleMethodName": func(method GoMethod) string {
				size := len(method.Returns)

				switch method.Type {
				case MethodJsonToJson:
					return fmt.Sprintf("handleJsonToJson%d", size)
				case MethodJsonToSSE:
					return "handleJsonToSSE"
				case MethodJsonToBinary:
					return "handleJsonToBinary"
				case MethodBinaryToJson:
					return fmt.Sprintf("handleBinaryToJson%d", size)
				case MethodBinaryToSSE:
					return "handleBinaryToSSE"
				case MethodBinaryToBinary:
					return "handleBinaryToBinary"
				default:
					panic(fmt.Sprintf("unknown method type: %d", method.Type))
				}
			},
			"ToMethodArgs": func(args []GoMethodArg) string {
				var sb strings.Builder

				sb.WriteString("ctx context.Context")

				for _, arg := range args {
					sb.WriteString(", ")
					sb.WriteString(arg.Name)
					sb.WriteString(" ")

					if arg.Stream && arg.Type == "[]byte" {
						sb.WriteString("func() (filename string, content io.Reader, err error)")
					} else {
						sb.WriteString(arg.Type)
					}
				}

				return sb.String()
			},
			"ToMethodReturns": func(returns []GoMethodReturn) string {
				var sb strings.Builder

				isChannel := false

				for i, ret := range returns {
					if i > 0 {
						sb.WriteString(", ")
					}

					sb.WriteString(ret.Name)
					sb.WriteString(" ")

					if ret.Stream && ret.Type != "[]byte" {
						sb.WriteString("<-chan ")
						sb.WriteString(ret.Type)
						isChannel = true
					} else if ret.Stream && ret.Type == "[]byte" {
						sb.WriteString("io.Reader")
					} else {
						sb.WriteString(ret.Type)
					}
				}

				if len(returns) > 0 {
					sb.WriteString(", ")
				}

				if isChannel {
					sb.WriteString("errs <-chan error")
				} else {
					sb.WriteString("err error")
				}

				return sb.String()
			},
			"ToMethodReturnTypeIndex": func(idx int, returns []GoMethodReturn) string {
				return returns[idx].Type
			},
			"HasOption": func(options []GoMethodOption, name string) bool {
				for _, opt := range options {
					if opt.Name == name {
						return true
					}
				}
				return false
			},
			"ToCallerResponse": func(returns []GoMethodReturn) string {
				var sb strings.Builder

				for _, ret := range returns {
					if ret.Type != "error" {
						sb.WriteString(", ")
						sb.WriteString(ret.Name)
					}
				}

				return sb.String()
			},
		}).
		ParseFS(golangTemplateFiles, "golang/*.go.tmpl")
	if err != nil {
		return err
	}

	out, err := os.Create(output)
	if err != nil {
		return err
	}

	// Helper functions

	isModelType := createIsModelTypeFunc(doc.Models)

	getServicesByType := func(typ ast.ServiceType) []GoService {
		return mapperFunc(getGolangServicesByType(doc.Services, typ), func(service *ast.Service) GoService {
			return GoService{
				Name: service.Name.Token.Value,
				Methods: mapperFunc(service.Methods, func(method *ast.Method) GoMethod {
					goMethod := GoMethod{
						Name:        method.Name.Token.Value,
						ServiceName: service.Name.Token.Value,
						Args: mapperFunc(method.Args, func(arg *ast.Arg) GoMethodArg {
							// func() (string, io.Reader, error)
							return GoMethodArg{
								Name:   strcase.ToCamel(arg.Name.Token.Value),
								Type:   getGolangType(arg.Type, isModelType),
								Stream: arg.Stream,
							}
						}),
						Returns: mapperFunc(method.Returns, func(ret *ast.Return) GoMethodReturn {
							// io.Reader
							return GoMethodReturn{
								Name:   strcase.ToCamel(ret.Name.Token.Value),
								Type:   getGolangType(ret.Type, isModelType),
								Stream: ret.Stream,
							}
						}),
						Options: mapperFunc(method.Options.List, func(opt *ast.Option) GoMethodOption {
							return GoMethodOption{
								Name:  opt.Name.Token.Value,
								Value: opt.Value,
							}
						}),
					}

					// Findout the method type
					// NOTE: currently stream keyword can at most appear once in the arguments and returns
					// if it appears more than once, it will be syntax error

					var argStreamType string
					var retStreamType string

					for _, arg := range goMethod.Args {
						if arg.Stream {
							argStreamType = arg.Type
							break
						}
					}

					for _, ret := range goMethod.Returns {
						if ret.Stream {
							retStreamType = ret.Type
							break
						}
					}

					if argStreamType == "" && retStreamType == "" {
						goMethod.Type = MethodJsonToJson
					} else if argStreamType == "" && retStreamType == "[]byte" {
						goMethod.Type = MethodJsonToBinary
					} else if argStreamType == "" && retStreamType != "" {
						goMethod.Type = MethodJsonToSSE
					} else if argStreamType != "" && retStreamType == "" {
						goMethod.Type = MethodBinaryToJson
					} else if argStreamType != "" && retStreamType == "[]byte" {
						goMethod.Type = MethodBinaryToBinary
					} else if argStreamType != "" && retStreamType != "" {
						goMethod.Type = MethodBinaryToSSE
					}

					return goMethod
				}),
			}
		})
	}

	data := Data{
		PackageName: pkg,
		Constants: mapperFunc(doc.Consts, func(c *ast.Const) GoConst {
			return GoConst{
				Name:  c.Identifier.Token.Value,
				Value: getGolangValue(c.Value),
			}
		}),
		Enums: mapperFunc(doc.Enums, func(enum *ast.Enum) GoEnum {
			return GoEnum{
				Name: enum.Name.Token.Value,
				Type: fmt.Sprintf("int%d", enum.Size),
				Keys: mapperFunc(enum.Sets, func(set *ast.EnumSet) GoEnumKeyValue {
					return GoEnumKeyValue{
						Name:  set.Name.Token.Value,
						Value: fmt.Sprintf("%d", set.Value.Value),
					}
				}),
			}
		}),
		Models: mapperFunc(doc.Models, func(model *ast.Model) GoModel {
			return GoModel{
				Name: model.Name.Token.Value,
				Fields: mapperFunc(model.Fields, func(field *ast.Field) GoModelField {
					return GoModelField{
						Name: field.Name.Token.Value,
						Type: getGolangType(field.Type, isModelType),
						Tags: getGolangModelFieldTag(field),
					}
				}),
			}
		}),
		HttpServices: getServicesByType(ast.ServiceHTTP),
		RpcServices:  getServicesByType(ast.ServiceRPC),
		Errors: mapperFunc(doc.Errors, func(err *ast.CustomError) GoError {
			return GoError{
				Name:    err.Name.Token.Value,
				Code:    err.Code,
				Message: err.Msg.Value,
			}
		}),
		Json2Json:   newSet[int](),
		Binary2Json: newSet[int](),
	}

	// adding some info about process functions
	// so they can be generated in the correct order
	for _, service := range data.HttpServices {
		for _, method := range service.Methods {
			switch method.Type {
			case MethodJsonToJson:
				data.Json2Json.add(len(method.Returns))
			case MethodJsonToBinary:
				data.Json2Binary = true
			case MethodJsonToSSE:
				data.Json2SSE = true
			case MethodBinaryToJson:
				data.Binary2Json.add(len(method.Returns))
			case MethodBinaryToBinary:
				data.Binary2Binary = true
			case MethodBinaryToSSE:
				data.Binary2SSE = true
			}
		}
	}

	// Eventhough Rpc methods currently can't have stream, but in the feature
	// adaptors can be added to support stream methods other than HTTP
	for _, service := range data.RpcServices {
		for _, method := range service.Methods {
			switch method.Type {
			case MethodJsonToJson:
				data.Json2Json.add(len(method.Returns))
			case MethodJsonToBinary:
				data.Json2Binary = true
			case MethodJsonToSSE:
				data.Json2SSE = true
			case MethodBinaryToJson:
				data.Binary2Json.add(len(method.Returns))
			case MethodBinaryToBinary:
				data.Binary2Binary = true
			case MethodBinaryToSSE:
				data.Binary2SSE = true
			}
		}
	}

	return tmpl.ExecuteTemplate(out, "main", data)
}

func getGolangServicesByType(services []*ast.Service, typ ast.ServiceType) []*ast.Service {
	return filterFunc(services, func(service *ast.Service) bool {
		return service.Type == typ
	})
}

func getGolangValue(value ast.Value) string {
	var sb strings.Builder

	switch v := value.(type) {
	case *ast.ValueString:
		if v.Token.Type == token.ConstStringSingleQuote {
			return fmt.Sprintf(`"%s"`, strings.ReplaceAll(v.Token.Value, `"`, `\"`))
		} else {
			value.Format(&sb)
			return sb.String()
		}
	case *ast.ValueInt:
		return strconv.FormatInt(v.Value, 10)
	case *ast.ValueByteSize:
		return fmt.Sprintf(`%d`, v.Value*int64(v.Scale))
	case *ast.ValueDuration:
		return fmt.Sprintf(`%d`, v.Value*int64(v.Scale))
	default:
		value.Format(&sb)
		return sb.String()
	}
}

func getGolangType(typ ast.Type, isModelType func(value string) bool) string {
	switch typ := typ.(type) {
	case *ast.CustomType:
		var sb strings.Builder
		typ.Format(&sb)
		val := sb.String()
		if isModelType(val) {
			return "*" + val
		}
		return val
	case *ast.Any:
		return "any"
	case *ast.Int:
		return fmt.Sprintf("int%d", typ.Size)
	case *ast.Uint:
		return fmt.Sprintf("uint%d", typ.Size)
	case *ast.Byte:
		return "byte"
	case *ast.Float:
		return fmt.Sprintf("float%d", typ.Size)
	case *ast.String:
		return "string"
	case *ast.Bool:
		return "bool"
	case *ast.Timestamp:
		return "time.Time"
	case *ast.Map:
		return fmt.Sprintf("map[%s]%s", getGolangType(typ.Key, isModelType), getGolangType(typ.Value, isModelType))
	case *ast.Array:
		return fmt.Sprintf("[]%s", getGolangType(typ.Type, isModelType))
	default:
		// This shouldn't happen as the validator should catch this any errors
		panic(fmt.Sprintf("unknown type: %T", typ))
	}
}

func getGolangModelFieldTag(field *ast.Field) string {
	var sb strings.Builder

	mapper := make(map[string]ast.Value)
	for _, opt := range field.Options.List {
		mapper[strings.ToLower(opt.Name.Token.Value)] = opt.Value
	}

	jsonTagValue := strcase.ToCamel(field.Name.Token.Value)

	jsonValue, ok := mapper["json"]
	if ok {
		switch jsonValue := jsonValue.(type) {
		case *ast.ValueString:
			jsonTagValue = jsonValue.Token.Value
		case *ast.ValueBool:
			if !jsonValue.Value {
				jsonTagValue = "-"
			}
		}
	}

	jsonOmitEmptyValue, isJsonOmitEmpty := mapper["jsonomitempty"]
	if isJsonOmitEmpty && jsonTagValue != "-" {
		switch value := jsonOmitEmptyValue.(type) {
		case *ast.ValueBool:
			if value.Value {
				jsonTagValue += ",omitempty"
			}
		}
	}

	if field.IsOptional {
		if !isJsonOmitEmpty {
			jsonTagValue += ",omitempty"
		}
		jsonTagValue += ",omitzero"
	}

	sb.WriteString(`json:"`)
	sb.WriteString(jsonTagValue)
	sb.WriteString(`"`)

	return sb.String()
}
