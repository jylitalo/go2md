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
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
)

// Markdown is golang template for go2md output
//
//go:embed template.md
var Markdown string // value from template.md file

func filter(info fs.FileInfo) bool {
	if strings.HasSuffix(info.Name(), "_test.go") {
		return false
	}
	if strings.HasSuffix(info.Name(), ".go") {
		return true
	}
	return false
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
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".go") {
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

// Run reads all "*.go" files (excluding "*_test.go") and writes markdown document out of it.
func Run(out io.Writer, directory, version string) error {
	fset := token.NewFileSet()
	modName, err := getPackageName(directory)
	if err != nil {
		return fmt.Errorf("unable to determine module name")
	}
	if !fileExists(directory + "/doc.go") {
		log.Warning("doc.go is missing")
	}
	astPackages, err := parser.ParseDir(fset, ".", filter, parser.ParseComments)
	if err != nil {
		return err
	}
	imports := dirImports(astPackages)
	imports["main"] = modName
	tmpl, err := template.New("new").Funcs(templateFuncs(version, imports)).Parse(Markdown)
	if err != nil {
		log.Error("Error from tmpl.New")
		return err
	}
	for _, astPkg := range astPackages {
		pkg := doc.New(astPkg, ".", 0)
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
