package main

import (
	"embed"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"

	"ella.to/ella/internal/strcase"
)

//go:embed templates/golang/*.go.tmpl
var golangTemplateFiles embed.FS

func generateGo(pkg, output string, doc *Document) error {
	// CONSTANTS

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
		Name string
		Type string
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
		Type        string // http, rpc
		ServiceName string // add this so it would be easier to generate the service path
		Args        []GoMethodArg
		Returns     []GoMethodReturn
		Options     []GoMethodOption

		HasArgs        bool
		HasReturns     bool
		HttpRawControl bool
		HttpMethod     string
		IsBinary       bool
		IsStream       bool
		IsUpload       bool
		Timeout        int64
		TotalMaxSize   int64
	}

	type GoService struct {
		Name    string
		Methods []GoMethod
	}

	// ERRORS

	type GoError struct {
		Name    string
		Code    int64
		Status  int
		Message string
	}

	type Data struct {
		PackageName  string
		Constants    []GoConst
		Enums        []GoEnum
		Models       []GoModel
		HttpServices []GoService
		RpcServices  []GoService
		Errors       []GoError
	}

	tmpl, err := template.
		New("GenerateGo").
		Funcs(defaultFuncsMap).
		Funcs(template.FuncMap{
			"ToMethodArgs": func(args []GoMethodArg) string {
				var sb strings.Builder

				sb.WriteString("ctx context.Context")

				// for file upload we will suffle the order of the arguments
				// and put the file upload argument at the end
				{
					// First find the index of the file upload argument
					fileUploadArgIdx := -1
					for i, arg := range args {
						if arg.Type == "func() (string, io.Reader, error)" {
							fileUploadArgIdx = i
							break
						}
					}

					// If we found the file upload argument, we will suffle the order
					// and move it to the end
					if fileUploadArgIdx != -1 {
						element := args[fileUploadArgIdx]
						// Remove the element from its current position
						args = append(args[:fileUploadArgIdx], args[fileUploadArgIdx+1:]...)
						// Append it to the end
						args = append(args, element)
					}
				}

				for _, arg := range args {
					sb.WriteString(", ")
					sb.WriteString(arg.Name)
					sb.WriteString(" ")
					sb.WriteString(arg.Type)
				}

				return sb.String()
			},
			"ToMethodReturns": func(returns []GoMethodReturn) string {
				var sb strings.Builder

				for i, ret := range returns {
					if i > 0 {
						sb.WriteString(", ")
					}

					sb.WriteString(ret.Name)
					sb.WriteString(" ")
					sb.WriteString(ret.Type)
				}

				if len(returns) > 0 {
					sb.WriteString(", ")
				}

				sb.WriteString("err error")

				return sb.String()
			},
			"HasOption": func(options []GoMethodOption, name string) bool {
				for _, opt := range options {
					if opt.Name == name {
						return true
					}
				}
				return false
			},
			"ToServicePathName": func(service GoService) string {
				return fmt.Sprintf("PathHttp%sPrefix", strcase.ToPascal(service.Name))
			},
			"ToServicePathValue": func(service GoService) string {
				return fmt.Sprintf("/ella/http/%s/", strcase.ToPascal(service.Name))
			},
			"ToMethodPathName": func(method GoMethod) string {
				return fmt.Sprintf("PathHttp%s%sMethod", strcase.ToPascal(method.ServiceName), strcase.ToPascal(method.Name))
			},
			"ToMethodPathValue": func(method GoMethod) string {
				return fmt.Sprintf("/ella/http/%s/%s", strcase.ToPascal(method.ServiceName), strcase.ToPascal(method.Name))
			},
			"ToHttpServiceImplName": func(service GoService) string {
				return fmt.Sprintf("http%sServer", strcase.ToPascal(service.Name))
			},
			"ToRpcServiceTopicName": func(service GoService) string {
				return fmt.Sprintf("TopicRpc%s", strcase.ToPascal(service.Name))
			},
			"ToRpcServiceTopicValue": func(service GoService) string {
				return fmt.Sprintf("ella.rpc.%s.*", strcase.ToSnake(service.Name))
			},
			"ToRpcServiceMethodTopicName": func(method GoMethod) string {
				return fmt.Sprintf("TopicRpc%s%sMethod", strcase.ToPascal(method.ServiceName), strcase.ToPascal(method.Name))
			},
			"ToRpcServiceMethodTopicValue": func(method GoMethod) string {
				return fmt.Sprintf("ella.rpc.%s.%s", strcase.ToSnake(method.ServiceName), strcase.ToSnake((method.Name)))
			},
			"ToArgsDefinition": func(tabs int, args []GoMethodArg) string {
				var sb strings.Builder

				i := 0

				for _, arg := range args {
					if arg.Type == "func() (string, io.Reader, error)" {
						continue
					}

					if i > 0 {
						sb.WriteString("\n")
					}

					sb.WriteString(strings.Repeat("	", tabs))
					sb.WriteString(strcase.ToPascal(arg.Name))
					sb.WriteString(" ")
					sb.WriteString(arg.Type)
					sb.WriteString(" `json:\"")
					sb.WriteString(strcase.ToCamel(arg.Name))
					sb.WriteString("\"`")

					i++
				}

				return sb.String()
			},
			"GetArgFileUploadName": func(args []GoMethodArg) string {
				for _, arg := range args {
					if arg.Type == "func() (string, io.Reader, error)" {
						return arg.Name
					}
				}

				return ""
			},
			"ArgsList": func(args []GoMethodArg) string {
				var sb strings.Builder

				i := 0
				for _, arg := range args {
					if arg.Type == "func() (string, io.Reader, error)" {
						continue
					}

					if i > 0 {
						sb.WriteString(", ")
					}

					sb.WriteString(arg.Name)
					i++
				}

				return sb.String()
			},
			"ToArgsAccess": func(prefix string, args []GoMethodArg) string {
				var sb strings.Builder

				sb.WriteString("ctx")

				i := 0
				for _, arg := range args {
					if arg.Type == "func() (string, io.Reader, error)" {
						continue
					}

					sb.WriteString(", ")

					sb.WriteString(prefix)
					sb.WriteString(strcase.ToPascal(arg.Name))
					i++
				}

				return sb.String()
			},
			"ToReturnsDefinition": func(tabs int, returns []GoMethodReturn) string {
				// We don't need to return anything if the method is a stream
				// because the return type is already known as (io.Reader, string, string, error)
				// for binary or channel of the return type
				for _, ret := range returns {
					if ret.Stream || strings.Index(ret.Type, "<-chan") != -1 {
						return ""
					}
				}

				var sb strings.Builder

				for i, ret := range returns {
					if i > 0 {
						sb.WriteString("\n")
					}

					sb.WriteString(strings.Repeat("	", tabs))
					sb.WriteString(strcase.ToPascal(ret.Name))
					sb.WriteString(" ")
					sb.WriteString(ret.Type)
					sb.WriteString(" `json:\"")
					sb.WriteString(strcase.ToCamel(ret.Name))
					sb.WriteString("\"`")

					i++
				}

				return sb.String()
			},
			"ToReturnsAccess": func(prefix string, returns []GoMethodReturn) string {
				var sb strings.Builder

				for i, ret := range returns {
					if i > 0 {
						sb.WriteString(", ")
					}

					sb.WriteString(prefix)
					sb.WriteString(strcase.ToPascal(ret.Name))
				}

				if len(returns) > 0 {
					sb.WriteString(", ")
				}

				sb.WriteString("err")

				return sb.String()
			},
			"StreamType": func(returns []GoMethodReturn) (typ string) {
				ret := returns[0]
				typ = strings.ReplaceAll(ret.Type, "<-chan ", "")

				return
			},
			"IsPointerType": func(typ string) bool {
				return strings.HasPrefix(typ, "*")
			},
		}).
		ParseFS(golangTemplateFiles, "templates/golang/*.go.tmpl")
	if err != nil {
		return err
	}

	out, err := os.Create(output)
	if err != nil {
		return err
	}

	// Helper functions

	isModelType := createIsModelTypeFunc(doc.Models)

	getServicesByMethodType := func(typ MethodType) []GoService {
		return mapperFunc(getGolangServicesByType(doc.Services, typ), func(service *Service) GoService {
			return GoService{
				Name: service.Name.Token.Value,
				Methods: mapperFunc(service.Methods, func(method *Method) GoMethod {
					goMethod := GoMethod{
						Name:        method.Name.Token.Value,
						Type:        method.Type.String(),
						ServiceName: service.Name.Token.Value,
						Args: mapperFunc(method.Args, func(arg *Arg) GoMethodArg {
							return GoMethodArg{
								Name: strcase.ToCamel(arg.Name.Token.Value),
								Type: getGolangType(arg.Type, isModelType),
							}
						}),
						Returns: mapperFunc(method.Returns, func(ret *Return) GoMethodReturn {
							typ := getGolangType(ret.Type, isModelType)
							if ret.Stream && typ == "[]byte" {
								typ = "io.Reader"
							}

							return GoMethodReturn{
								Name:   strcase.ToCamel(ret.Name.Token.Value),
								Type:   typ,
								Stream: ret.Stream,
							}
						}),
						Options: mapperFunc(method.Options.List, func(opt *Option) GoMethodOption {
							return GoMethodOption{
								Name:  opt.Name.Token.Value,
								Value: opt.Value,
							}
						}),
					}

					goMethod.HasArgs = len(goMethod.Args) > 0
					goMethod.HasReturns = len(goMethod.Returns) > 0

					// the default value for http method is POST
					if typ == MethodHTTP {
						goMethod.HttpMethod = "POST"
						goMethod.TotalMaxSize = 2 * 1024 * 1024
					}

					for _, opt := range goMethod.Options {
						switch opt.Name {
						case "HttpRawControl":
							if typ != MethodHTTP {
								break
							}

							goMethod.HttpRawControl = true

						case "HttpMethod":
							if typ != MethodHTTP {
								break
							}

							if v, ok := opt.Value.(*ValueString); ok {
								goMethod.HttpMethod = v.Value
							} else {
								goMethod.HttpMethod = "POST"
							}

						case "Timeout":
							if typ != MethodHTTP {
								break
							}

							if v, ok := opt.Value.(*ValueDuration); ok {
								goMethod.Timeout = v.Value * int64(v.Scale)
							}

						case "TotalMaxSize":
							if typ != MethodHTTP {
								break
							}

							if v, ok := opt.Value.(*ValueByteSize); ok {
								goMethod.TotalMaxSize = v.Value * int64(v.Scale)
							}
						}
					}

					for _, arg := range goMethod.Args {
						if arg.Type == "func() (string, io.Reader, error)" {
							if typ == MethodHTTP {
								goMethod.IsUpload = true
							}
						}
					}

					for _, ret := range goMethod.Returns {
						if ret.Stream {
							if typ == MethodHTTP {
								goMethod.IsStream = true
							}
						}

						if ret.Type == "io.Reader" {
							if typ == MethodHTTP {
								goMethod.IsBinary = true
							}
						}
					}

					if goMethod.IsStream && !goMethod.IsBinary {
						goMethod.Returns = []GoMethodReturn{
							{
								Name: goMethod.Returns[0].Name,
								Type: "<-chan " + goMethod.Returns[0].Type,
							},
						}
					}

					// need to add filename:string and contentType:string to returns
					// if the method is stream and binary
					if goMethod.IsStream && goMethod.IsBinary {
						goMethod.Returns = append(goMethod.Returns, GoMethodReturn{
							Name: "filename",
							Type: "string",
						}, GoMethodReturn{
							Name: "contentType",
							Type: "string",
						})
					}

					return goMethod
				}),
			}
		})
	}

	data := Data{
		PackageName: pkg,
		Constants: mapperFunc(doc.Consts, func(c *Const) GoConst {
			return GoConst{
				Name:  c.Identifier.Token.Value,
				Value: getGolangValue(c.Value),
			}
		}),
		Enums: mapperFunc(doc.Enums, func(enum *Enum) GoEnum {
			return GoEnum{
				Name: enum.Name.Token.Value,
				Type: fmt.Sprintf("int%d", enum.Size),
				Keys: mapperFunc(enum.Sets, func(set *EnumSet) GoEnumKeyValue {
					return GoEnumKeyValue{
						Name:  set.Name.Token.Value,
						Value: fmt.Sprintf("%d", set.Value.Value),
					}
				}),
			}
		}),
		Models: mapperFunc(doc.Models, func(model *Model) GoModel {
			return GoModel{
				Name: model.Name.Token.Value,
				Fields: mapperFunc(model.Fields, func(field *Field) GoModelField {
					return GoModelField{
						Name: field.Name.Token.Value,
						Type: getGolangType(field.Type, isModelType),
						Tags: getGolangModelFieldTag(field),
					}
				}),
			}
		}),
		HttpServices: getServicesByMethodType(MethodHTTP),
		RpcServices:  getServicesByMethodType(MethodRPC),
		Errors: mapperFunc(doc.Errors, func(err *CustomError) GoError {
			return GoError{
				Name:    err.Name.Token.Value,
				Code:    err.Code,
				Status:  err.HttpStatus,
				Message: err.Msg.Value,
			}
		}),
	}

	return tmpl.ExecuteTemplate(out, "main", data)
}

func getGolangServicesByType(services []*Service, typ MethodType) []*Service {
	return mapperFunc(filterFunc(services, func(service *Service) bool {
		for _, method := range service.Methods {
			if method.Type == typ || method.Type == MethodRpcHttp {
				return true
			}
		}
		return false
	}), func(service *Service) *Service {
		return &Service{
			Token: service.Token,
			Name:  service.Name,
			Methods: filterFunc(service.Methods, func(method *Method) bool {
				return method.Type == typ || method.Type == MethodRpcHttp
			}),
			Comments: service.Comments,
		}
	})
}

func getGolangValue(value Value) string {
	var sb strings.Builder

	switch v := value.(type) {
	case *ValueString:
		if v.Token.Type == TokConstStringSingleQuote {
			return fmt.Sprintf(`"%s"`, strings.ReplaceAll(v.Token.Value, `"`, `\"`))
		} else {
			value.Format(&sb)
			return sb.String()
		}
	case *ValueInt:
		return strconv.FormatInt(v.Value, 10)
	case *ValueByteSize:
		return fmt.Sprintf(`%d`, v.Value*int64(v.Scale))
	case *ValueDuration:
		return fmt.Sprintf(`%d`, v.Value*int64(v.Scale))
	default:
		value.Format(&sb)
		return sb.String()
	}
}

func getGolangType(typ Type, isModelType func(value string) bool) string {
	switch typ := typ.(type) {
	case *CustomType:
		var sb strings.Builder
		typ.Format(&sb)
		val := sb.String()
		if isModelType(val) {
			return "*" + val
		}
		return val
	case *Any:
		return "any"
	case *Int:
		return fmt.Sprintf("int%d", typ.Size)
	case *Uint:
		return fmt.Sprintf("uint%d", typ.Size)
	case *Byte:
		return "byte"
	case *Float:
		return fmt.Sprintf("float%d", typ.Size)
	case *String:
		return "string"
	case *Bool:
		return "bool"
	case *Timestamp:
		return "time.Time"
	case *Map:
		return fmt.Sprintf("map[%s]%s", getGolangType(typ.Key, isModelType), getGolangType(typ.Value, isModelType))
	case *Array:
		return fmt.Sprintf("[]%s", getGolangType(typ.Type, isModelType))
	case *File:
		return "func() (string, io.Reader, error)"
	}

	// This shouldn't happen as the validator should catch this any errors
	panic(fmt.Sprintf("unknown type: %T", typ))
}

func getGolangModelFieldTag(field *Field) string {
	var sb strings.Builder

	mapper := make(map[string]Value)
	for _, opt := range field.Options.List {
		mapper[strings.ToLower(opt.Name.Token.Value)] = opt.Value
	}

	jsonTagValue := strcase.ToCamel(field.Name.Token.Value)

	jsonValue, ok := mapper["json"]
	if ok {
		switch jsonValue := jsonValue.(type) {
		case *ValueString:
			jsonTagValue = jsonValue.Token.Value
		case *ValueBool:
			if !jsonValue.Value {
				jsonTagValue = "-"
			}
		}
	}

	jsonOmitEmptyValue, ok := mapper["jsonomitempty"]
	if ok && jsonTagValue != "-" {
		switch value := jsonOmitEmptyValue.(type) {
		case *ValueBool:
			if value.Value {
				jsonTagValue += ",omitempty"
			}
		}
	}

	sb.WriteString(`json:"`)
	sb.WriteString(jsonTagValue)
	sb.WriteString(`"`)

	return sb.String()
}
