package main

import (
	"sort"

	"ella.to/ella/internal/strcase"
)

// Checks the following
// [x] All the names should be camelCase and PascalCase
// [x] All the names should be unique (const, model, enum and services)
// [x] All the same service's method names should be unique
// [x] All the same enum's keys should be unique
// [x] Constant assignment should be valid and the name of the constant should be available
// [x] Check if Custom Types (Model and Enum names) are defined in Model's fields and Service's arguments and return types
// [ ] There should be only one method's argument with type of file
// [ ] There should be only one stream return type
// [ ] The key type of map should be comparable type
// [ ] Validate if Custom Error Code and HttpStatus are valid

func Validate(docs ...*Document) error {
	consts := make([]*Const, 0)
	enums := make([]*Enum, 0)
	models := make([]*Model, 0)
	services := make([]*Service, 0)
	customErrors := make([]*CustomError, 0)

	// Since all the ella's documents are compiled into a single file,
	// First we need to sort all the consts, enums, models and services

	for _, doc := range docs {
		for _, c := range doc.Consts {
			consts = append(consts, c)
		}

		for _, e := range doc.Enums {
			enums = append(enums, e)
		}

		for _, m := range doc.Models {
			models = append(models, m)
		}

		for _, s := range doc.Services {
			services = append(services, s)
		}

		for _, e := range doc.Errors {
			customErrors = append(customErrors, e)
		}
	}

	{
		// check for CamelCase names
		for _, c := range consts {
			if !strcase.IsPascal(c.Identifier.Token.Value) {
				return NewError(c.Identifier.Token, "name should be PascalCase")
			}
		}

		for _, e := range enums {
			if !strcase.IsPascal(e.Name.Token.Value) {
				return NewError(e.Name.Token, "name should be PascalCase")
			}

			for _, k := range e.Sets {
				if k.Name.Token.Value == "_" {
					continue
				}

				if !strcase.IsPascal(k.Name.Token.Value) {
					return NewError(k.Name.Token, "name should be PascalCase")
				}
			}
		}

		for _, m := range models {
			if !strcase.IsPascal(m.Name.Token.Value) {
				return NewError(m.Name.Token, "name should be PascalCase")
			}

			for _, f := range m.Fields {
				if !strcase.IsPascal(f.Name.Token.Value) {
					return NewError(f.Name.Token, "name should be PascalCase")
				}

				for _, o := range f.Options.List {
					if !strcase.IsPascal(o.Name.Token.Value) {
						return NewError(o.Name.Token, "name should be PascalCase")
					}
				}
			}
		}

		for _, s := range services {
			if !strcase.IsPascal(s.Name.Token.Value) {
				return NewError(s.Name.Token, "name should be PascalCase")
			}

			for _, m := range s.Methods {
				if !strcase.IsPascal(m.Name.Token.Value) {
					return NewError(m.Name.Token, "name should be PascalCase")
				}

				for _, a := range m.Args {
					if !strcase.IsCamel(a.Name.Token.Value) {
						return NewError(a.Name.Token, "name should be camelCase")
					}
				}

				for _, r := range m.Returns {
					if !strcase.IsCamel(r.Name.Token.Value) {
						return NewError(r.Name.Token, "name should be camelCase")
					}
				}

				for _, o := range m.Options.List {
					if !strcase.IsPascal(o.Name.Token.Value) {
						return NewError(o.Name.Token, "name should be PascalCase")
					}
				}
			}
		}
	}

	{
		// check for duplicate names

		duplicateNames := make(map[string]struct{})
		for _, c := range consts {
			if _, ok := duplicateNames[c.Identifier.Token.Value]; ok {
				return NewError(c.Identifier.Token, "name is already used")
			}
			duplicateNames[c.Identifier.Token.Value] = struct{}{}
		}

		for _, e := range enums {
			if _, ok := duplicateNames[e.Name.Token.Value]; ok {
				return NewError(e.Name.Token, "name is already used")
			}
			duplicateNames[e.Name.Token.Value] = struct{}{}

			enumDuplicateKeys := make(map[string]struct{})
			for _, k := range e.Sets {
				if k.Name.Token.Value == "_" {
					continue
				}

				if _, ok := enumDuplicateKeys[k.Name.Token.Value]; ok {
					return NewError(k.Name.Token, "key is already used in the same enum")
				}
				enumDuplicateKeys[k.Name.Token.Value] = struct{}{}
			}
		}

		for _, m := range models {
			if _, ok := duplicateNames[m.Name.Token.Value]; ok {
				return NewError(m.Name.Token, "name is already used")
			}
			duplicateNames[m.Name.Token.Value] = struct{}{}

			modelDuplicateFields := make(map[string]struct{})
			for _, f := range m.Fields {
				if _, ok := modelDuplicateFields[f.Name.Token.Value]; ok {
					return NewError(f.Name.Token, "field name is already used in the same model")
				}
				modelDuplicateFields[f.Name.Token.Value] = struct{}{}

				modelOptionDuplicateNames := make(map[string]struct{})
				for _, o := range f.Options.List {
					if _, ok := modelOptionDuplicateNames[o.Name.Token.Value]; ok {
						return NewError(o.Name.Token, "option name is already used in the same field")
					}
					modelOptionDuplicateNames[o.Name.Token.Value] = struct{}{}
				}
			}
		}

		for _, s := range services {
			if _, ok := duplicateNames[s.Name.Token.Value]; ok {
				return NewError(s.Name.Token, "name is already used")
			}
			duplicateNames[s.Name.Token.Value] = struct{}{}

			serviceDuplicateMethods := make(map[string]struct{})
			for _, m := range s.Methods {
				if _, ok := serviceDuplicateMethods[m.Name.Token.Value]; ok {
					return NewError(m.Name.Token, "method name is already used in the same service")
				}
				serviceDuplicateMethods[m.Name.Token.Value] = struct{}{}

				serviceMethodDuplicateArguments := make(map[string]struct{})
				for _, a := range m.Args {
					if _, ok := serviceMethodDuplicateArguments[a.Name.Token.Value]; ok {
						return NewError(a.Name.Token, "argument name is already used in the same method")
					}
					serviceMethodDuplicateArguments[a.Name.Token.Value] = struct{}{}
				}

				serviceMethodDuplicateReturns := make(map[string]struct{})
				for _, r := range m.Returns {
					if _, ok := serviceMethodDuplicateReturns[r.Name.Token.Value]; ok {
						return NewError(r.Name.Token, "return name is already used in the same method")
					}
					serviceMethodDuplicateReturns[r.Name.Token.Value] = struct{}{}

					if _, ok := serviceMethodDuplicateArguments[r.Name.Token.Value]; ok {
						return NewError(r.Name.Token, "return name is already used in the same method as argument")
					}
				}

				serviceMethodDuplicateOptions := make(map[string]struct{})
				for _, o := range m.Options.List {
					if _, ok := serviceMethodDuplicateOptions[o.Name.Token.Value]; ok {
						return NewError(o.Name.Token, "option name is already used in the same method")
					}
					serviceMethodDuplicateOptions[o.Name.Token.Value] = struct{}{}
				}
			}
		}

		{
			constMap := make(map[string]*Const)

			for _, c := range consts {
				constMap[c.Identifier.Token.Value] = c
			}

			var findConstValue func(name string) Value
			findConstValue = func(name string) Value {
				c, ok := constMap[name]
				if !ok {
					return nil
				}

				if v, ok := c.Value.(*ValueVariable); ok {
					return findConstValue(v.Token.Value)
				}

				return c.Value
			}

			for _, c := range consts {
				if variable, ok := c.Value.(*ValueVariable); ok {
					value := findConstValue(variable.Token.Value)
					if value == nil {
						return NewError(variable.Token, "unknown constant is not defined")
					}
					c.Value = value
				}
			}

			for _, m := range models {
				for _, f := range m.Fields {
					for _, o := range f.Options.List {
						if variable, ok := o.Value.(*ValueVariable); ok {
							value := findConstValue(variable.Token.Value)
							if value == nil {
								return NewError(variable.Token, "unknown constant is not defined")
							}
							o.Value = value
						}
					}
				}
			}

			for _, s := range services {
				for _, m := range s.Methods {
					for _, o := range m.Options.List {
						if variable, ok := o.Value.(*ValueVariable); ok {
							value := findConstValue(variable.Token.Value)
							if value == nil {
								return NewError(variable.Token, "unknown constant is not defined")
							}
							o.Value = value
						}
					}
				}
			}
		}
	}

	{
		// check for custom types name exist
		typesMap := make(map[string]struct{})

		for _, m := range models {
			typesMap[m.Name.Token.Value] = struct{}{}
		}

		for _, e := range enums {
			typesMap[e.Name.Token.Value] = struct{}{}
		}

		// check for custom types name exist in models
		for _, m := range models {
			for _, f := range m.Fields {
				if err := checkTypeExists(typesMap, f.Type); err != nil {
					return err
				}
			}
		}

		// check for custom types name exist in services
		for _, s := range services {
			for _, m := range s.Methods {
				for _, a := range m.Args {
					if err := checkTypeExists(typesMap, a.Type); err != nil {
						return err
					}
				}

				for _, r := range m.Returns {
					if err := checkTypeExists(typesMap, r.Type); err != nil {
						return err
					}
				}
			}
		}
	}

	{
		// check for custom errors
		sort.Slice(customErrors, func(i, j int) bool {
			return customErrors[i].Name.Token.Value < customErrors[j].Name.Token.Value
		})

		var maxCode int64 = 0
		var reservedCodes = make(map[int64]struct{})
		for _, e := range customErrors {
			if _, ok := reservedCodes[e.Code]; ok {
				return NewError(e.Token, "code is already used")
			}
			if e.Code != 0 {
				reservedCodes[e.Code] = struct{}{}
				maxCode = max(maxCode, e.Code)
			}
		}

		for _, e := range customErrors {
			if e.Code == 0 {
				maxCode++
				e.Code = maxCode
			}
		}

		for _, e := range customErrors {
			if _, ok := HttpStatusCode2String[e.HttpStatus]; !ok {
				return NewError(e.Token, "http status is not valid in custom error")
			}
		}
	}

	return nil
}

func checkTypeExists(typesMap map[string]struct{}, t Type) error {
	switch v := t.(type) {
	case *Map:
		return checkTypeExists(typesMap, v.Value)
	case *Array:
		return checkTypeExists(typesMap, v.Type)
	case *CustomType:
		if _, ok := typesMap[v.Token.Value]; !ok {
			return NewError(v.Token, "type is not defined")
		}
		return nil
	default:
		// Handle other types which is already checked in the parser
		return nil
	}
}
