package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"ella.to/ella/compiler"
)

const Version = "0.3.0"

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
        ella fmt [--debug] <glob path>

  - gen Generate code from a folder to a file.
        Supports: .go, _js.go (WASM bindings), .d.ts, or .ts (TypeScript)
        ella gen [--debug] <pkg> <output path to file> <search glob paths...>

  - ver Print the version of ella

Flags:
  --debug  Print the AST (Abstract Syntax Tree) for debugging
  --allow-ext  Enable extension registration for *_js.go generation

Output file conventions:
  *.go       Generate Go code (models, services, clients)
  *_js.go    Generate WASM/JS bindings for browser
  *.d.ts     Generate TypeScript definitions for wasm

Examples:
  ella fmt "./path/to/*.ella"
  ella fmt --debug "./path/to/*.ella"
  ella gen schema ./path/to/schema_gen.go "./path/to/*.ella"
  ella gen schema --allow-ext ./path/to/schema_gen_js.go "./path/to/*.ella"
  ella gen schema ./path/to/schema_gen_js.go "./path/to/*.ella"
  ella gen schema ./path/to/schema.d.ts "./path/to/*.ella"
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(0)
	}

	var err error
	var files []string

	defer func() {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}()

	cmd := os.Args[1]
	argc := len(os.Args)

	switch cmd {
	case "fmt":
		if argc < 3 {
			fmt.Print(usage)
			os.Exit(0)
		}

		debug := false
		paths := os.Args[2:]

		// Check for --debug flag
		if len(paths) > 0 && paths[0] == "--debug" {
			debug = true
			paths = paths[1:]
		}

		if len(paths) == 0 {
			fmt.Print(usage)
			os.Exit(0)
		}

		files, err = getFilesByGlob(paths...)
		if err != nil {
			return
		}

		formatCmd(files, debug)

	case "gen":
		if argc < 5 {
			fmt.Print(usage)
			os.Exit(0)
		}

		debug := false
		allowExt := false
		rawArgs := os.Args[2:]
		args := make([]string, 0, len(rawArgs))

		for _, arg := range rawArgs {
			switch arg {
			case "--debug":
				debug = true
			case "--allow-ext":
				allowExt = true
			default:
				if strings.HasPrefix(arg, "--") {
					showErrors(fmt.Errorf("unknown flag: %s", arg))
					return
				}
				args = append(args, arg)
			}
		}

		if len(args) < 3 {
			fmt.Print(usage)
			os.Exit(0)
		}

		pkg := args[0]
		out := args[1]
		paths := args[2:]

		files, err = getFilesByGlob(paths...)
		if err != nil {
			return
		}

		genCmd(files, pkg, out, debug, allowExt)

	case "ver":
		fmt.Println(Version)

	default:
		fmt.Print(usage)
		os.Exit(0)
	}
}

func formatCmd(ins []string, debug bool) {
	runner := NewGoroutineLimiter(runtime.NumCPU())

	for _, in := range ins {
		runner.Run(func() error {
			file, err := os.Open(in)
			if err != nil {
				return err
			}
			defer file.Close()

			prog, err := compiler.NewParser(compiler.NewScanner(file, in)).Parse()
			if err != nil {
				return err
			}

			if debug {
				printAST(in, prog)
			}

			formatted := compiler.Format(prog)

			return os.WriteFile(in, []byte(formatted), os.ModePerm)
		})
	}

	errs := runner.Wait()
	if len(errs) > 0 {
		showErrors(errs...)
		return
	}
}

func genCmd(ins []string, pkg string, out string, debug bool, allowExt bool) {
	runner := NewGoroutineLimiter(runtime.NumCPU())
	programs := make([]*compiler.Program, len(ins))

	for i, in := range ins {
		runner.Run(func() error {
			file, err := os.Open(in)
			if err != nil {
				return err
			}
			defer file.Close()

			prog, err := compiler.NewParser(compiler.NewScanner(file, in)).Parse()
			if err != nil {
				return err
			}

			if debug {
				printAST(in, prog)
			}

			programs[i] = prog

			return nil
		})
	}

	errs := runner.Wait()
	if len(errs) > 0 {
		showErrors(errs...)
		return
	}

	// merge programs
	var prog compiler.Program
	for _, program := range programs {
		prog.Nodes = append(prog.Nodes, program.Nodes...)
		prog.Comments = append(prog.Comments, program.Comments...)
	}

	if debug {
		fmt.Println("\n=== Merged Program AST ===")
		printAST("merged", &prog)
	}

	errs = compiler.ValidateProgram(&prog)
	if len(errs) > 0 {
		showErrors(errs...)
		return
	}

	// Determine output type based on file extension
	switch {
	case strings.HasSuffix(out, ".d.ts"):
		// TypeScript declaration generation
		genTypeScript(out, &prog)

		// Emit runtime values into a sibling .ts file when schema has consts or errors.
		if hasRuntimeTypeScriptExports(&prog) {
			runtimeOut := strings.TrimSuffix(out, ".d.ts") + ".ts"
			genTypeScriptRuntimeConsts(runtimeOut, &prog)
		}
	case strings.HasSuffix(out, ".ts"):
		// TypeScript runtime client generation
		genTypeScriptClient(out, &prog)
	case strings.HasSuffix(out, "_js.go"):
		// WASM bindings generation (also generates the base .go file)
		baseOut := strings.TrimSuffix(out, "_js.go") + ".go"
		genGoCode(baseOut, pkg, &prog)
		genWasmCode(out, pkg, &prog, allowExt)
	case strings.HasSuffix(out, ".go"):
		// Standard Go generation
		genGoCode(out, pkg, &prog)
	default:
		showErrors(fmt.Errorf("unsupported output file extension: %s (use .go, _js.go, .ts, or .d.ts)", out))
	}
}

func hasConstDeclarations(prog *compiler.Program) bool {
	for _, node := range prog.Nodes {
		if _, ok := node.(*compiler.ConstDecl); ok {
			return true
		}
	}

	return false
}

func hasErrorDeclarations(prog *compiler.Program) bool {
	for _, node := range prog.Nodes {
		if _, ok := node.(*compiler.DeclError); ok {
			return true
		}
	}

	return false
}

func hasRuntimeTypeScriptExports(prog *compiler.Program) bool {
	return hasConstDeclarations(prog) || hasErrorDeclarations(prog)
}

func genGoCode(out string, pkg string, prog *compiler.Program) {
	f, err := os.Create(out)
	if err != nil {
		showErrors(err)
		return
	}
	defer f.Close()

	gen := compiler.NewGoGenerator(prog, pkg)

	err = gen.GenerateToWriter(f)
	if err != nil {
		showErrors(err)
		return
	}

	// add helper functions
	_, err = f.WriteString(gen.GenerateHelperTypes())
	if err != nil {
		showErrors(err)
		return
	}
}

func genWasmCode(out string, pkg string, prog *compiler.Program, allowExt bool) {
	wasmFile, err := os.Create(out)
	if err != nil {
		showErrors(err)
		return
	}
	defer wasmFile.Close()

	wasmGen := compiler.NewWasmGenerator(prog, pkg, allowExt)
	err = wasmGen.GenerateToWriter(wasmFile)
	if err != nil {
		showErrors(err)
		return
	}
}

func genTypeScript(out string, prog *compiler.Program) {
	tsFile, err := os.Create(out)
	if err != nil {
		showErrors(err)
		return
	}
	defer tsFile.Close()

	tsGen := compiler.NewTypeScriptGenerator(prog)
	err = tsGen.GenerateToWriter(tsFile)
	if err != nil {
		showErrors(err)
		return
	}
}

func genTypeScriptClient(out string, prog *compiler.Program) {
	tsFile, err := os.Create(out)
	if err != nil {
		showErrors(err)
		return
	}
	defer tsFile.Close()

	tsGen := compiler.NewTypeScriptGenerator(prog)
	err = tsGen.GenerateClientToWriter(tsFile)
	if err != nil {
		showErrors(err)
		return
	}
}

func genTypeScriptRuntimeConsts(out string, prog *compiler.Program) {
	tsFile, err := os.Create(out)
	if err != nil {
		showErrors(err)
		return
	}
	defer tsFile.Close()

	tsGen := compiler.NewTypeScriptGenerator(prog)
	err = tsGen.GenerateRuntimeConstsToWriter(tsFile)
	if err != nil {
		showErrors(err)
		return
	}
}

func showErrors(errs ...error) {
	for _, err := range errs {
		switch e := err.(type) {
		case *compiler.Error:
			src := e.Token.Pos.Src
			b, readErr := os.ReadFile(src)
			if readErr == nil {
				fmt.Println(compiler.NewErrorDisplay(string(b), src).FormatCompilerError(e))
			} else {
				// Fallback: print the error without source context
				fmt.Printf("error: %s\n", e.Reason)
				if src != "" {
					fmt.Printf("  --> %s:%d:%d\n", src, e.Token.Pos.Line, e.Token.Pos.Column)
				} else {
					fmt.Printf("  --> line %d, column %d\n", e.Token.Pos.Line, e.Token.Pos.Column)
				}
			}
		default:
			fmt.Println(err)
		}
	}
}

func printAST(filename string, prog *compiler.Program) {
	fmt.Printf("\n=== AST for %s ===\n", filename)
	fmt.Printf("Nodes: %d, Comments: %d\n\n", len(prog.Nodes), len(prog.Comments))

	for i, node := range prog.Nodes {
		fmt.Printf("[%d] %T\n", i, node)
		printNodeDetails(node, "  ")
	}

	if len(prog.Comments) > 0 {
		fmt.Printf("\nComments:\n")
		for i, comment := range prog.Comments {
			fmt.Printf("  [%d] Line %d: %s\n", i, comment.Pos.Line, comment.Lit)
		}
	}
	fmt.Println()
}

func printNodeDetails(node compiler.Node, indent string) {
	switch n := node.(type) {
	case *compiler.ConstDecl:
		fmt.Printf("%sName: %s\n", indent, n.Assignment.Name.Name)
		fmt.Printf("%sValue: %T = %s\n", indent, n.Assignment.Value, n.Assignment.Value.String())
	case *compiler.DeclEnum:
		fmt.Printf("%sName: %s\n", indent, n.Name.Name)
		fmt.Printf("%sValues:\n", indent)
		for _, v := range n.Values {
			if v.IsDefined {
				fmt.Printf("%s  - %s = %s\n", indent, v.Name.Name, v.Value.String())
			} else {
				fmt.Printf("%s  - %s\n", indent, v.Name.Name)
			}
		}
	case *compiler.DeclModel:
		fmt.Printf("%sName: %s\n", indent, n.Name.Name)
		if len(n.Extends) > 0 {
			fmt.Printf("%sExtends:\n", indent)
			for _, ext := range n.Extends {
				fmt.Printf("%s  - %s\n", indent, ext.Name)
			}
		}
		fmt.Printf("%sFields:\n", indent)
		for _, f := range n.Fields {
			fmt.Printf("%s  - %s: %s\n", indent, f.Name.Name, f.Type.String())
		}
	case *compiler.DeclService:
		fmt.Printf("%sName: %s\n", indent, n.Name.Name)
		fmt.Printf("%sMethods:\n", indent)
		for _, m := range n.Methods {
			fmt.Printf("%s  - %s\n", indent, m.Name.Name)
			if len(m.Args) > 0 {
				fmt.Printf("%s    Args:\n", indent)
				for _, arg := range m.Args {
					fmt.Printf("%s      - %s: %s\n", indent, arg.Name.Name, arg.Type.String())
				}
			}
			if len(m.Returns) > 0 {
				fmt.Printf("%s    Returns:\n", indent)
				for _, ret := range m.Returns {
					fmt.Printf("%s      - %s: %s\n", indent, ret.Name.Name, ret.Type.String())
				}
			}
		}
	case *compiler.DeclError:
		fmt.Printf("%sName: %s\n", indent, n.Name.Name)
		if n.Code != nil {
			fmt.Printf("%sCode: %s\n", indent, n.Code.String())
		}
		fmt.Printf("%sMsg: %s\n", indent, n.Msg.String())
	}
}

// make sure only pattern is used at the end of the search path
// and only one level of search path is allowed
func getFilesByGlob(searchPaths ...string) ([]string, error) {
	filenames := []string{}

	for _, searchPath := range searchPaths {
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
	}

	return filenames, nil
}

// Helper functions for parallel parsing
// GoroutineLimiter controls the maximum number of concurrent goroutines
type GoroutineLimiter struct {
	sem    chan struct{}
	wg     sync.WaitGroup
	errsMu sync.Mutex
	errs   []error
}

// NewGoroutineLimiter creates a new limiter with the specified max goroutines
func NewGoroutineLimiter(maxGoroutines int) *GoroutineLimiter {
	return &GoroutineLimiter{
		sem:  make(chan struct{}, maxGoroutines),
		errs: make([]error, 0),
	}
}

// Run executes the given function, blocking if max goroutines reached
func (g *GoroutineLimiter) Run(fn func() error) {
	g.wg.Add(1)
	g.sem <- struct{}{} // Acquire semaphore (blocks if full)

	go func() {
		defer func() {
			<-g.sem // Release semaphore
			g.wg.Done()
		}()

		if err := fn(); err != nil {
			g.errsMu.Lock()
			g.errs = append(g.errs, err)
			g.errsMu.Unlock()
		}
	}()
}

// Wait blocks until all running goroutines complete and returns all errors
func (g *GoroutineLimiter) Wait() []error {
	g.wg.Wait()
	return g.errs
}
