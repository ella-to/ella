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

//go:embed templates/typescript/*.ts.tmpl
var typescriptTemplateFiles embed.FS

func generateTypescript(pkg, output string, doc *Document) error {

	// Note: Currently we only care about the http services
	// in typescript, so we filter out the rpc services.
	doc.Services = filterFunc(doc.Services, func(service *Service) bool {
		return service.Token.Type != TokenType(ServiceRPC)
	})

	// CONSTANTS

	type TsConst struct {
		Name  string
		Value string
	}

	// ENUMS

	type TsEnumKeyValue struct {
		Name  string
		Value string
	}

	type TsEnum struct {
		Name string
		Keys []TsEnumKeyValue
	}

	// MODELS

	type TsField struct {
		Name       string
		Type       string
		IsOptional bool
	}

	type TsModel struct {
		Name   string
		Fields []TsField
	}

	// SERVICES

	type TsArg struct {
		Name string
		Type string
	}

	type TsReturn struct {
		Name string
		Type string
	}

	type TsMethod struct {
		Name        string
		ServiceName string
		Type        string // normal, raw, binary, stream, fileupload
		HttpMethod  string // GET, POST, PUT, DELETE, PATCH, OPTIONS
		Args        []TsArg
		Returns     []TsReturn
	}

	type TsService struct {
		Name    string
		Methods []TsMethod
	}

	// CUSTOM ERROR

	type TsError struct {
		Name string
		Code int64
	}

	// Data

	type Data struct {
		PackageName  string
		Constants    []TsConst
		Enums        []TsEnum
		Models       []TsModel
		HttpServices []TsService
		Errors       []TsError
	}

	data := Data{
		PackageName: pkg,
		Constants: mapperFunc(doc.Consts, func(c *Const) TsConst {
			return TsConst{
				Name:  c.Identifier.Token.Value,
				Value: getGolangValue(c.Value),
			}
		}),
		Enums: mapperFunc(doc.Enums, func(enum *Enum) TsEnum {
			return TsEnum{
				Name: enum.Name.Token.Value,
				Keys: mapperFunc(filterFunc(enum.Sets, func(set *EnumSet) bool {
					return set.Name.Token.Value != "_"
				}), func(set *EnumSet) TsEnumKeyValue {
					return TsEnumKeyValue{
						Name:  set.Name.Token.Value,
						Value: strcase.ToSnake(set.Name.Token.Value),
					}
				}),
			}
		}),
		Models: mapperFunc(doc.Models, func(model *Model) TsModel {
			return TsModel{
				Name: model.Name.Token.Value,
				Fields: filterFunc(mapperFunc(model.Fields, func(field *Field) TsField {
					name := strcase.ToSnake(field.Name.Token.Value)
					for _, opt := range field.Options.List {
						if opt.Name.Token.Value == "Json" {
							switch v := opt.Value.(type) {
							case *ValueString:
								name = v.Value
							case *ValueBool:
								if !v.Value {
									name = ""
								}
							}
							break
						}
					}

					return TsField{
						Name:       name,
						Type:       getTypescriptType(field.Type),
						IsOptional: field.IsOptional,
					}
				}), func(field TsField) bool {
					return field.Name != ""
				}),
			}
		}),
		HttpServices: mapperFunc(doc.Services, func(service *Service) TsService {
			return TsService{
				Name: service.Name.Token.Value,
				Methods: mapperFunc(service.Methods, func(method *Method) TsMethod {
					var tsMethod TsMethod

					tsMethod.Name = method.Name.Token.Value
					tsMethod.ServiceName = service.Name.Token.Value
					tsMethod.Args = filterFunc(mapperFunc(method.Args, func(arg *Arg) TsArg {
						return TsArg{
							Name: arg.Name.Token.Value,
							Type: getTypescriptType(arg.Type),
						}
					}), func(ta TsArg) bool {
						if ta.Type == "fileupload" {
							tsMethod.Type = "fileupload"
						}

						return ta.Type != "fileupload"
					})
					tsMethod.Returns = mapperFunc(method.Returns, func(ret *Return) TsReturn {
						typ := getTypescriptType(ret.Type)

						if ret.Stream && typ == "byte[]" {
							tsMethod.Type = "binary"
						} else if ret.Stream {
							tsMethod.Type = "stream"
						} else if tsMethod.Type != "fileupload" {
							tsMethod.Type = "normal"
						}

						return TsReturn{
							Name: ret.Name.Token.Value,
							Type: typ,
						}
					})

					if tsMethod.Type == "" {
						for _, opt := range method.Options.List {
							if opt.Name.Token.Value == "HttpRawControl" {
								tsMethod.Type = "raw"
								break
							}
						}
					}

					tsMethod.HttpMethod = "POST"
					for _, opt := range method.Options.List {
						if opt.Name.Token.Value == "HttpMethod" {
							tsMethod.HttpMethod = opt.Value.(*ValueString).Value
							break
						}
					}

					return tsMethod
				}),
			}
		}),
		Errors: mapperFunc(doc.Errors, func(err *CustomError) TsError {
			return TsError{
				Name: err.Name.Token.Value,
				Code: err.Code,
			}
		}),
	}

	tmpl, err := template.
		New("GenerateTS").
		Funcs(defaultFuncsMap).
		Funcs(template.FuncMap{
			"ArgsName": func(method TsMethod) string {
				return fmt.Sprintf("Service%s%sArgs", method.ServiceName, strcase.ToPascal(method.Name))
			},
			"ReturnsName": func(method TsMethod) string {
				switch method.Type {
				case "binary":
					return "Blob"
				case "stream":
					return "Subscription<" + method.Returns[0].Type + ">"
				default:
					return fmt.Sprintf("Service%s%sReturns", method.ServiceName, strcase.ToPascal(method.Name))
				}
			},
			"ShouldGenerateReturn": func(method TsMethod) bool {
				return method.Type != "stream" && method.Type != "binary"
			},
			"MethodPathValue": func(method TsMethod) string {
				return fmt.Sprintf("/ella/http/%s/%s", strcase.ToPascal(method.ServiceName), strcase.ToPascal(method.Name))
			},
		}).
		ParseFS(typescriptTemplateFiles, "templates/typescript/*.ts.tmpl")
	if err != nil {
		return err
	}

	out, err := os.Create(output)
	if err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(out, "main", data)
}

func getTypescriptValue(value Value) string {
	switch v := value.(type) {
	case *ValueString:
		if v.Token.Type == TokConstStringSingleQuote {
			return fmt.Sprintf(`"%s"`, strings.ReplaceAll(v.Token.Value, `"`, `\"`))
		} else {
			var sb strings.Builder
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
		var sb strings.Builder
		value.Format(&sb)
		return sb.String()
	}
}

func getTypescriptType(typ Type) string {
	switch t := typ.(type) {
	case *Bool:
		return `boolean`
	case *Int, *Float, *Uint:
		return `number`
	case *String:
		return `string`
	case *Any:
		return `any`
	case *Timestamp:
		return `string`
	case *Array:
		typ := getTypescriptType(t.Type)
		return typ + "[]"
	case *Map:
		key := getTypescriptType(t.Key)
		value := getTypescriptType(t.Value)
		return `{ [key: ` + key + `]: ` + value + ` }`
	case *CustomType:
		return t.Token.Value
	case *Byte:
		return "byte"
	default:
		panic(fmt.Errorf("unknown type: %T", t))
	}
}
