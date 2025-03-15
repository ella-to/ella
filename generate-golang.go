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

		IsBinary     bool
		IsStream     bool
		IsUpload     bool
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
		Status  int
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
		Process       map[int]struct{}
		ProcessStream bool
		ProcessBinary bool
		ProcessUpload map[int]struct{}
	}

	tmpl, err := template.
		New("GenerateGo").
		Funcs(defaultFuncsMap).
		Funcs(template.FuncMap{
			"GenArgsGenerics": func(size int) string {
				var sb strings.Builder

				sb.WriteString("A")
				for i := 1; i <= size; i++ {
					sb.WriteString(fmt.Sprintf(", R%d", i))
				}
				sb.WriteString(" any")

				return sb.String()
			},
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
			"ProcessName": func(method GoMethod) string {
				if method.IsUpload {
					return fmt.Sprintf("processUpload%d", len(method.Returns))
				}

				if method.IsBinary {
					return "processBinary"
				}

				if method.IsStream {
					return "processStream"
				}

				return fmt.Sprintf("process%d", len(method.Returns))
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

				for i, ret := range returns {
					if i > 0 {
						sb.WriteString(", ")
					}

					sb.WriteString(ret.Name)
					sb.WriteString(" ")

					if ret.Stream && ret.Type != "[]byte" {
						sb.WriteString("<-chan ")
						sb.WriteString(ret.Type)
					} else if ret.Stream && ret.Type == "[]byte" {
						sb.WriteString("io.Reader")
					} else {
						sb.WriteString(ret.Type)
					}
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
		}).
		ParseFS(golangTemplateFiles, "templates/golang/*.go.tmpl")
	if err != nil {
		return err
	}

	// List all the files inside golangTemplateFiles
	//
	// fmt.Println("Golang template files:")
	// entries, err := golangTemplateFiles.ReadDir("templates/golang")
	// if err != nil {
	// 	return fmt.Errorf("failed to read template directory: %w", err)
	// }

	// for _, entry := range entries {
	// 	if !entry.IsDir() {
	// 		fmt.Printf("  - %s\n", entry.Name())
	// 	}
	// }

	out, err := os.Create(output)
	if err != nil {
		return err
	}

	// Helper functions

	isModelType := createIsModelTypeFunc(doc.Models)

	getServicesByType := func(typ ServiceType) []GoService {
		return mapperFunc(getGolangServicesByType(doc.Services, typ), func(service *Service) GoService {
			return GoService{
				Name: service.Name.Token.Value,
				Methods: mapperFunc(service.Methods, func(method *Method) GoMethod {
					goMethod := GoMethod{
						Name:        method.Name.Token.Value,
						ServiceName: service.Name.Token.Value,
						Args: mapperFunc(method.Args, func(arg *Arg) GoMethodArg {
							// func() (string, io.Reader, error)
							return GoMethodArg{
								Name:   strcase.ToCamel(arg.Name.Token.Value),
								Type:   getGolangType(arg.Type, isModelType),
								Stream: arg.Stream,
							}
						}),
						Returns: mapperFunc(method.Returns, func(ret *Return) GoMethodReturn {
							// io.Reader
							return GoMethodReturn{
								Name:   strcase.ToCamel(ret.Name.Token.Value),
								Type:   getGolangType(ret.Type, isModelType),
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

					for _, arg := range goMethod.Args {
						if arg.Stream {
							goMethod.IsUpload = true
							break
						}
					}

					for _, ret := range goMethod.Returns {
						if ret.Stream {
							goMethod.IsStream = true
						}

						if ret.Type == "[]byte" {
							goMethod.IsBinary = true
						}
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
		HttpServices: getServicesByType(ServiceHTTP),
		RpcServices:  getServicesByType(ServiceRPC),
		Errors: mapperFunc(doc.Errors, func(err *CustomError) GoError {
			return GoError{
				Name:    err.Name.Token.Value,
				Code:    err.Code,
				Status:  err.HttpStatus,
				Message: err.Msg.Value,
			}
		}),
		Process:       make(map[int]struct{}),
		ProcessStream: false,
		ProcessBinary: false,
		ProcessUpload: make(map[int]struct{}),
	}

	// adding some info about process functions
	// so they can be generated in the correct order
	for _, service := range data.HttpServices {
		for _, method := range service.Methods {
			if method.IsUpload {
				data.ProcessUpload[len(method.Returns)] = struct{}{}
			} else if method.IsBinary {
				data.ProcessBinary = true
			} else if method.IsStream {
				data.ProcessStream = true
			} else {
				data.Process[len(method.Returns)] = struct{}{}
			}
		}
	}
	for _, service := range data.HttpServices {
		for _, method := range service.Methods {
			data.Process[len(method.Returns)] = struct{}{}
		}
	}

	return tmpl.ExecuteTemplate(out, "main", data)
}

func getGolangServicesByType(services []*Service, typ ServiceType) []*Service {
	return filterFunc(services, func(service *Service) bool {
		return service.Type == typ
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
	default:
		// This shouldn't happen as the validator should catch this any errors
		panic(fmt.Sprintf("unknown type: %T", typ))
	}
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

	jsonOmitEmptyValue, isJsonOmitEmpty := mapper["jsonomitempty"]
	if isJsonOmitEmpty && jsonTagValue != "-" {
		switch value := jsonOmitEmptyValue.(type) {
		case *ValueBool:
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
