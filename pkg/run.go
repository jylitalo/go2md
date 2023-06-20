package pkg

import (
	_ "embed"
	"errors"
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

type OutputSettings struct {
	Default   io.WriteCloser // current default
	Directory string         // override Out with Directory + Filename
	Filename  string         // override Out with Directory + Filename
}

type lineNumber struct {
	filename string
	line     int
}

type packageInfo struct {
	pkg         doc.Package
	imports     map[string]string
	lineNumbers map[string]lineNumber
}

var (
	// Markdown is golang template for go2md output
	//
	//go:embed template.md
	Markdown          string // value from template.md file
	ErrNoPackageFound = errors.New("couldn't find package from ")
)

// isProductionGo ignores all code that is only used for `go test`
func isProductionGo(filename string) bool {
	return strings.HasSuffix(filename, ".go") && !strings.HasSuffix(filename, "_test.go")
}

// buildImportMap creates map from all imports that one directory has.
// Key is alias to package and value is full path to package.
// If alias is `_`, we ignore it.
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

// dirImports scan through all ast.Packages and creates map of import statements.
// Key is alias to package and value is full path to package.
// If alias is `_`, we ignore it.
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
func (output *OutputSettings) Writer() (io.WriteCloser, error) {
	if output.Filename == "" {
		return output.Default, nil
	}
	fname := output.Directory + "/" + output.Filename
	fout, err := os.Create(fname)
	if err != nil {
		return output.Default, fmt.Errorf("OutputSettings.Writer failed: %w", err)
	}
	return fout, err
}

// RunDirectory checks given directory and only that directory
// Returns ErrNoPackagesFound if includeMain=true and current directory has only main package.
func RunDirectory(out OutputSettings, version string, includeMain bool) error {
	pkgName, err := getPackageName(out.Directory)
	if err != nil {
		return fmt.Errorf("failed determine module name: %w", err)
	}
	return run(out, pkgName, version, includeMain)
}

// RunDirTree checks given directory and its subdirectories with RunDirectory().
// Ignores all ErrNoPackageFound errors from RunDirectory.
func RunDirTree(out OutputSettings, version string, includeMain bool) error {
	paths := map[string]bool{}
	err := filepath.WalkDir(out.Directory, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && strings.HasSuffix(path, ".go") {
			paths[filepath.Dir(path)] = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	for path := range paths {
		out.Directory = path
		if err = RunDirectory(out, version, includeMain); err != nil {
			if errors.Is(err, ErrNoPackageFound) {
				log.Warning("failed to find package from " + path)
				continue
			}
			return err
		}
	}
	return nil
}

// isExported checks if given pattern (typically var, const, type or function name)
// can be used from other packages.
func isExported(pattern string) bool {
	ok, _ := regexp.Match("^[A-Z]", []byte(pattern))
	return ok
}

// scanFile goes through given file and builds map that gives line number for every
// function and type in a file.
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

// getPackage reads all golang code into ast.Packages,
// parses information about imported packages,
// functions linenumbers in golang files and
// at the end converts ast.Package into doc.Package.
// If directory has references to more than one package, that is error because
// multiple packages would overwrite each others output.
// If includeMain is false and directory has main package, it returns ErrNoPackageFound
func getPackage(directory, modName string, includeMain bool) (*packageInfo, error) {
	retValue := &packageInfo{
		lineNumbers: map[string]lineNumber{},
	}
	pkgs := []doc.Package{}
	fset := token.NewFileSet()
	if !fileExists(directory + "/doc.go") {
		log.Warning("doc.go is missing from " + directory)
	}
	astPackages, err := parser.ParseDir(fset, directory, func(fi fs.FileInfo) bool {
		fname := directory + "/" + fi.Name()
		if valid := isProductionGo(fname); !valid {
			return false
		}
		for key, value := range scanFile(fname) {
			retValue.lineNumbers[key] = lineNumber{filename: fi.Name(), line: value}
		}
		return true
	}, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	retValue.imports = dirImports(astPackages)
	for _, astPkg := range astPackages {
		pkg := doc.New(astPkg, directory, 0)
		if pkg.Name == "main" && !includeMain {
			log.Warningf("Ignoring main package due to --ignore-main")
			continue
		}
		// log.WithFields(log.Fields{"pkg": fmt.Sprintf("%#v", pkg)}).Info("output from doc.New")
		// log.WithFields(log.Fields{"pkg.Types": fmt.Sprintf("%#v: %#v", fset.Position(token.Pos(pkg.Types[0].Decl.Tok)), pkg.Types[0])}).Info("output from doc.New")
		if strings.HasSuffix(modName, "/"+pkg.Name) {
			pkg.Name = modName
		}
		pkgs = append(pkgs, *pkg)
	}
	switch len(pkgs) {
	case 0:
		return nil, fmt.Errorf("%w %s", ErrNoPackageFound, directory)
	case 1:
		retValue.pkg = pkgs[0]
		return retValue, nil
	}
	names := []string{}
	for _, pkg := range pkgs {
		names = append(names, pkg.Name)
	}
	return nil, fmt.Errorf("can only handle one package per directory (found: %s)", strings.Join(names, ", "))
}

// Run reads all "*.go" files (excluding "*_test.go") and writes markdown document out of it.
func run(out OutputSettings, modName, version string, includeMain bool) error {
	pkgInfo, err := getPackage(out.Directory, modName, includeMain)
	if err != nil {
		return fmt.Errorf("getPackages failed: %w", err)
	}
	pkgInfo.imports["main"] = modName
	tmpl, err := template.New("new").Funcs(templateFuncs(version, pkgInfo.imports, pkgInfo.lineNumbers)).Parse(Markdown)
	if err != nil {
		return fmt.Errorf("tmpl.New failed: %w", err)
	}
	if pkgInfo.pkg.Name == "main" && !includeMain {
		return nil
	}
	writer, err := out.Writer()
	if err != nil {
		return err
	}
	defer writer.Close()
	if err = tmpl.Execute(writer, pkgInfo.pkg); err != nil {
		return fmt.Errorf("tmpl.Execute failed: %w", err)
	}
	return nil
}
