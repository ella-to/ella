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

//go:embed typescript/*.ts.tmpl
var typescriptTemplateFiles embed.FS

func generateTypescript(pkg, output string, doc *ast.Document) error {
	// Note: Currently we only care about the http services
	// in typescript, so we filter out the rpc services.
	doc.Services = filterFunc(doc.Services, func(service *ast.Service) bool {
		return service.Token.Type != token.Type(ast.ServiceRPC)
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
		Name   string
		Type   string
		Stream bool
	}

	type TsReturn struct {
		Name   string
		Type   string
		Stream bool
	}

	type TsMethod struct {
		Name        string
		ServiceName string
		ReqType     string // json, fileupload
		RespType    string // json, blob, sse
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
		Constants: mapperFunc(doc.Consts, func(c *ast.Const) TsConst {
			return TsConst{
				Name:  c.Identifier.Token.Value,
				Value: getGolangValue(c.Value),
			}
		}),
		Enums: mapperFunc(doc.Enums, func(enum *ast.Enum) TsEnum {
			return TsEnum{
				Name: enum.Name.Token.Value,
				Keys: mapperFunc(filterFunc(enum.Sets, func(set *ast.EnumSet) bool {
					return set.Name.Token.Value != "_"
				}), func(set *ast.EnumSet) TsEnumKeyValue {
					return TsEnumKeyValue{
						Name:  set.Name.Token.Value,
						Value: strcase.ToSnake(set.Name.Token.Value),
					}
				}),
			}
		}),
		Models: mapperFunc(doc.Models, func(model *ast.Model) TsModel {
			return TsModel{
				Name: model.Name.Token.Value,
				Fields: filterFunc(mapperFunc(model.Fields, func(field *ast.Field) TsField {
					name := strcase.ToSnake(field.Name.Token.Value)
					for _, opt := range field.Options.List {
						if opt.Name.Token.Value == "Json" {
							switch v := opt.Value.(type) {
							case *ast.ValueString:
								name = v.Value
							case *ast.ValueBool:
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
		HttpServices: mapperFunc(getServicesByType(doc.Services, ast.ServiceHTTP), func(service *ast.Service) TsService {
			return TsService{
				Name: service.Name.Token.Value,
				Methods: mapperFunc(service.Methods, func(method *ast.Method) TsMethod {
					var tsMethod TsMethod

					tsMethod.Name = method.Name.Token.Value
					tsMethod.ServiceName = service.Name.Token.Value
					tsMethod.Args = mapperFunc(
						method.Args,
						func(arg *ast.Arg) TsArg {
							return TsArg{
								Name:   arg.Name.Token.Value,
								Type:   getTypescriptType(arg.Type),
								Stream: arg.Stream,
							}
						},
					)
					tsMethod.Returns = mapperFunc(method.Returns, func(ret *ast.Return) TsReturn {
						return TsReturn{
							Name:   ret.Name.Token.Value,
							Type:   getTypescriptType(ret.Type),
							Stream: ret.Stream,
						}
					})

					tsMethod.ReqType = "JSON"

					for _, arg := range tsMethod.Args {
						if arg.Stream && arg.Type == "byte[]" {
							tsMethod.ReqType = "FILE_UPLOAD"
							break
						}
					}

					tsMethod.RespType = "JSON"

					for _, ret := range tsMethod.Returns {
						if ret.Stream {
							if ret.Type == "byte[]" {
								tsMethod.RespType = "BLOB"
								break
							}

							tsMethod.RespType = "SSE"
							break
						}
					}

					return tsMethod
				}),
			}
		}),
		Errors: mapperFunc(doc.Errors, func(err *ast.CustomError) TsError {
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
			"ToArgs": func(args []TsArg) string {
				var sb strings.Builder
				for i, arg := range args {
					if i > 0 {
						sb.WriteString(", ")
					}
					sb.WriteString(arg.Name)
					sb.WriteString(": ")

					if arg.Stream {
						sb.WriteString("fileData[]")
					} else {
						sb.WriteString(arg.Type)
					}
				}

				if sb.Len() > 0 {
					sb.WriteString(", ")
				}

				sb.WriteString("_opts?: reqOpts")

				return sb.String()
			},
			"ToParams": func(args []TsArg) string {
				var sb strings.Builder

				i := 0
				for _, arg := range args {
					if arg.Stream {
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
			// <subscription<Type>>
			// <Blob>
			// <[string, number, User]>
			"ToReturns": func(method TsMethod) string {
				if method.RespType == "SSE" {
					return fmt.Sprintf("subscription<%s>", method.Returns[0].Type)
				}

				if method.RespType == "BLOB" {
					return "Blob"
				}

				var sb strings.Builder

				sb.WriteString("[")
				for i, ret := range method.Returns {
					if i > 0 {
						sb.WriteString(", ")
					}
					sb.WriteString(ret.Type)
				}

				sb.WriteString("]")

				return sb.String()
			},
			"ToFileUploadArgName": func(args []TsArg) string {
				for _, arg := range args {
					if arg.Stream && arg.Type == "byte[]" {
						return arg.Name
					}
				}

				return "undefined"
			},
		}).
		ParseFS(typescriptTemplateFiles, "typescript/*.ts.tmpl")
	if err != nil {
		return err
	}

	out, err := os.Create(output)
	if err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(out, "main", data)
}

func getTypescriptValue(value ast.Value) string {
	switch v := value.(type) {
	case *ast.ValueString:
		if v.Token.Type == token.ConstStringSingleQuote {
			return fmt.Sprintf(`"%s"`, strings.ReplaceAll(v.Token.Value, `"`, `\"`))
		} else {
			var sb strings.Builder
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
		var sb strings.Builder
		value.Format(&sb)
		return sb.String()
	}
}

func getTypescriptType(typ ast.Type) string {
	switch t := typ.(type) {
	case *ast.Bool:
		return `boolean`
	case *ast.Int, *ast.Float, *ast.Uint:
		return `number`
	case *ast.String:
		return `string`
	case *ast.Any:
		return `any`
	case *ast.Timestamp:
		return `string`
	case *ast.Array:
		typ := getTypescriptType(t.Type)
		return typ + "[]"
	case *ast.Map:
		key := getTypescriptType(t.Key)
		value := getTypescriptType(t.Value)
		return `{ [key: ` + key + `]: ` + value + ` }`
	case *ast.CustomType:
		return t.Token.Value
	case *ast.Byte:
		return "byte"
	default:
		panic(fmt.Errorf("unknown type: %T", t))
	}
}
