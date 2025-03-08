package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const Version = "0.1.4"

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
		err = gen(os.Args[2], os.Args[3], os.Args[4:]...)
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

func format(searchPaths ...string) error {
	for _, searchPath := range searchPaths {
		filenames, err := filesFromGlob(searchPath)
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
	}

	return nil
}

func gen(pkg, out string, searchPaths ...string) (err error) {
	var docs []*Document

	for _, searchPath := range searchPaths {
		filenames, err := filesFromGlob(searchPath)
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

	return Generate(pkg, out, docs)
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

	fmt.Println("dir: ", dir, "pattern: ", pattern)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		fmt.Println(entry.Name(), entry.IsDir())
	}

	return filenames, nil

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

	for _, filename := range filenames {
		fmt.Println(filename)
	}

	return filenames, nil
}
