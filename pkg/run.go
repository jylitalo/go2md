package pkg

import (
	_ "embed"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
)

// Markdown is golang template for go2md output
//
//go:embed template.md
var Markdown string // value from template.md file

func isProductionGo(filename string) bool {
	return strings.HasSuffix(filename, ".go") && !strings.HasSuffix(filename, "_test.go")
}

func buildImportMap(specs []*ast.ImportSpec) map[string]string {
	mapping := map[string]string{}
	for _, spec := range specs {
		path := strings.Trim(spec.Path.Value, ("\""))
		switch {
		case spec.Name == nil:
			fields := strings.Split(path, "/")
			mapping[fields[len(fields)-1]] = path
		case spec.Name.Name == "_":
			continue
		default:
			mapping[spec.Name.Name] = path
		}
	}
	return mapping
}

func dirImports(astPackages map[string]*ast.Package) map[string]string {
	mapping := map[string]string{}
	stats := map[string]map[string][]string{}
	for _, astPkg := range astPackages {
		for fname, f := range astPkg.Files {
			for alias, fq := range buildImportMap(f.Imports) {
				mapping[alias] = fq
				if _, ok := stats[alias]; ok {
					if _, ok := stats[alias][fq]; ok {
						stats[alias][fq] = append(stats[alias][fq], fname)
						continue
					}
					stats[alias][fq] = []string{fname}
					continue
				}
				stats[alias] = map[string][]string{fq: {fname}}
			}
		}
	}
	for alias := range stats {
		if len(stats[alias]) == 1 {
			continue
		}
		files := []string{}
		for fq, fname := range stats[alias] {
			files = append(files, fmt.Sprintf("%s in %s", fq, strings.Join(fname, ", ")))
		}
		log.Warningf("%s has been imported as \n- %s", alias, strings.Join(files, "\n- "))
	}
	return mapping
}

// Output creates output file if needed and returns writer to it
func Output(out io.Writer, directory, filename string) (io.Writer, func() error, error) {
	if filename == "" {
		return out, nil, nil
	}
	fname := directory + "/" + filename
	fout, err := os.Create(fname)
	if err != nil {
		log.WithFields(log.Fields{"err": err, "fname": fname}).Fatal("failed to create file")
		return out, nil, err
	}
	return fout, fout.Close, err
}

// RunDirTree checks given directory and its subdirectories for golang
func RunDirTree(out io.Writer, directory, output, version string) error {
	paths := map[string]bool{}
	err := filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && strings.HasSuffix(path, ".go") {
			paths[filepath.Dir(path)] = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	for path := range paths {
		out, close, err := Output(out, path, output)
		if err != nil {
			return err
		}
		if close != nil {
			defer close()
		}
		if err = Run(out, path, version); err != nil {
			return err
		}
	}
	return nil
}

type lineNumber struct {
	filename string
	line     int
}

func isExported(pattern string) bool {
	ok, _ := regexp.Match("^[A-Z]", []byte(pattern))
	return ok
}

func scanFile(filename string) map[string]int {
	content, err := os.ReadFile(filename)
	if err != nil {
		log.WithFields(log.Fields{"err": err, "filename": filename}).Error("scanFile")
		return nil
	}
	lineNumbers := map[string]int{}
	for lineNumber, line := range strings.Split(string(content), "\n") {
		if strings.HasPrefix(line, "func ") {
			words := strings.Split(line, " ")
			if len(words) >= 4 && strings.HasPrefix(words[1], "(") && isExported(words[3]) {
				words[3] = strings.Split(words[3], "[")[0]
				words[3] = strings.Split(words[3], "(")[0]
				key := intoLink(strings.Join(words[0:4], " "))
				lineNumbers[key] = lineNumber + 1
			}
			if len(words) >= 2 && isExported(words[1]) {
				words[1] = strings.Split(words[1], "[")[0]
				words[1] = strings.Split(words[1], "(")[0]
				key := intoLink(strings.Join(words[0:2], " "))
				lineNumbers[key] = lineNumber + 1
			}
		}
		if strings.HasPrefix(line, "type ") {
			words := strings.Split(line, " ")
			if len(words) >= 2 && isExported(words[1]) {
				lineNumbers[intoLink(strings.Join(words[0:2], " "))] = lineNumber + 1
			}
		}
	}
	return lineNumbers
}

// Run reads all "*.go" files (excluding "*_test.go") and writes markdown document out of it.
func Run(out io.Writer, directory, version string) error {
	fset := token.NewFileSet()
	modName, err := getPackageName(directory)
	if err != nil {
		return fmt.Errorf("unable to determine module name")
	}
	if !fileExists(directory + "/doc.go") {
		log.Warning("doc.go is missing from " + directory)
	}
	lineNumbers := map[string]lineNumber{}
	astPackages, err := parser.ParseDir(fset, directory, func(fi fs.FileInfo) bool {
		fname := directory + "/" + fi.Name()
		if valid := isProductionGo(fname); !valid {
			return false
		}
		for key, value := range scanFile(fname) {
			lineNumbers[key] = lineNumber{filename: fname, line: value}
		}
		return true
	}, parser.ParseComments)
	if err != nil {
		return err
	}
	imports := dirImports(astPackages)
	imports["main"] = modName
	tmpl, err := template.New("new").Funcs(templateFuncs(version, imports, lineNumbers)).Parse(Markdown)
	if err != nil {
		log.Error("Error from tmpl.New")
		return err
	}
	for _, astPkg := range astPackages {
		pkg := doc.New(astPkg, directory, 0)
		// log.WithFields(log.Fields{"pkg": fmt.Sprintf("%#v", pkg)}).Info("output from doc.New")
		// log.WithFields(log.Fields{"pkg.Types": fmt.Sprintf("%#v: %#v", fset.Position(token.Pos(pkg.Types[0].Decl.Tok)), pkg.Types[0])}).Info("output from doc.New")
		if strings.HasSuffix(modName, "/"+pkg.Name) {
			pkg.Name = modName
		}
		if err = tmpl.Execute(out, pkg); err != nil {
			log.Error("Error from tmpl.Execute")
			return err
		}
	}
	return nil
}
