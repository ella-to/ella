package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const Version = "0.1.2"

const usage = `
███████╗██╗░░░░░██╗░░░░░░█████╗░
██╔════╝██║░░░░░██║░░░░░██╔══██╗
█████╗░░██║░░░░░██║░░░░░███████║
██╔══╝░░██║░░░░░██║░░░░░██╔══██║
███████╗███████╗███████╗██║░░██║
╚══════╝╚══════╝╚══════╝╚═╝░░╚═╝ v` + Version + `

Usage: ella [command]

Commands:
  - fmt Format one or many files in place using glob pattern
        ella fmt <glob path>

  - gen Generate code from a folder to a file and currently
        supports .go and .ts extensions
        ella gen <pkg> <output path to file> <search glob paths...>

  - ver Print the version of ella

example:
  ella fmt "./path/to/*.ella"
  ella gen rpc ./path/to/output.go "./path/to/*.ella"
  ella gen rpc ./path/to/output.ts "./path/to/*.ella" "./path/to/other/*.ella"
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(0)
	}

	var err error

	switch os.Args[1] {
	case "fmt":
		if len(os.Args) < 3 {
			fmt.Print(usage)
			os.Exit(0)
		}
		err = format(os.Args[2])
	case "gen":
		if len(os.Args) < 5 {
			fmt.Print(usage)
			os.Exit(0)
		}
		err = gen(os.Args[2], os.Args[3], os.Args[4:], nil)
	case "gen-models":
		if len(os.Args) < 5 {
			fmt.Print(usage)
			os.Exit(0)
		}

		pkg := os.Args[2]
		out := os.Args[3]
		searchPaths := os.Args[4:]

		err = genModelsOnly(pkg, out, searchPaths...)

	case "gen-services":
		if len(os.Args) < 7 {
			fmt.Print(usage)
			os.Exit(0)
		}

		pkg := os.Args[2]
		out := os.Args[3]
		modelsPkg := os.Args[4]
		modelsPath := os.Args[5]
		searchPaths := os.Args[6:]

		err = genServicesOnly(pkg, out, modelsPkg, modelsPath, searchPaths...)

	case "ver":
		fmt.Println(Version)
	default:
		fmt.Print(usage)
		os.Exit(0)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func format(path string) error {
	filenames, err := filepath.Glob(path)
	if err != nil {
		return err
	}

	for _, filename := range filenames {
		doc, err := ParseDocument(NewParserWithFilenames(filename))
		if err != nil {
			return err
		}

		var sb strings.Builder
		doc.Format(&sb)

		err = os.WriteFile(filename, []byte(sb.String()), os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

func gen(pkg, out string, searchPaths []string, opt *GenerateOption) (err error) {
	var docs []*Document

	for _, searchPath := range searchPaths {
		filenames, err := filepath.Glob(searchPath)
		if err != nil {
			return err
		}

		for _, filename := range filenames {
			doc, err := ParseDocument(NewParserWithFilenames(filename))
			if err != nil {
				return err
			}

			docs = append(docs, doc)
		}
	}

	if err = Validate(docs...); err != nil {
		return err
	}

	return Generate(pkg, out, docs, opt)
}

func genModelsOnly(pkg, out string, searchPaths ...string) (err error) {
	return gen(pkg, out, searchPaths, &GenerateOption{ModelsOnly: true})
}

func genServicesOnly(pkg, out, modelsPkg, modelsPath string, searchPaths ...string) (err error) {
	return gen(pkg, out, searchPaths, &GenerateOption{
		ServicesOnly: true,
		ModelPkgName: modelsPkg,
		ModelPkgPath: modelsPath,
	})
}
