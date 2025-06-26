package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ella.to/ella/internal/compiler/ast"
	"ella.to/ella/internal/compiler/gen"
	"ella.to/ella/internal/compiler/parser"
)

const Version = "0.2.6"

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
		err = formatCmd(os.Args[2])
	case "gen":
		if len(os.Args) < 5 {
			fmt.Print(usage)
			os.Exit(0)
		}
		err = genCmd(os.Args[2], os.Args[3], os.Args[4:]...)
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

func formatCmd(searchPaths ...string) error {
	for _, searchPath := range searchPaths {
		filenames, err := filesFromGlob(searchPath)
		if err != nil {
			return err
		}

		for _, filename := range filenames {
			doc, err := parser.ParseDocument(parser.NewWithFilenames(filename))
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
	}

	return nil
}

func genCmd(pkg, out string, searchPaths ...string) (err error) {
	var docs []*ast.Document

	for _, searchPath := range searchPaths {
		filenames, err := filesFromGlob(searchPath)
		if err != nil {
			return err
		}

		for _, filename := range filenames {
			doc, err := parser.ParseDocument(parser.NewWithFilenames(filename))
			if err != nil {
				return err
			}

			docs = append(docs, doc)
		}
	}

	if err = parser.Validate(docs...); err != nil {
		return err
	}

	return gen.Generate(pkg, out, docs)
}

// make sure only pattern is used at the end of the search path
// and only one level of search path is allowed
func filesFromGlob(searchPath string) ([]string, error) {
	filenames := []string{}

	dir, pattern := filepath.Split(searchPath)
	if dir == "" {
		dir = "."
	}

	if strings.Contains(dir, "*") {
		return nil, fmt.Errorf("glob pattern should not be used in dir level: %s", searchPath)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		match, err := filepath.Match(pattern, entry.Name())
		if err != nil {
			return nil, err
		}
		if match {
			filenames = append(filenames, filepath.Join(dir, entry.Name()))
		}
	}

	return filenames, nil
}
