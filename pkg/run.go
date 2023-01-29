package pkg

import (
	_ "embed"
	"fmt"
	"go/doc"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
)

//go:embed template.md
var Markdown string

func filter(info fs.FileInfo) bool {
	if strings.HasSuffix(info.Name(), "_test.go") {
		return false
	}
	if strings.HasSuffix(info.Name(), ".go") {
		return true
	}
	return false
}

func Run(version string) error {
	fset := token.NewFileSet()
	modName, err := getPackageName(".")
	if err != nil {
		return fmt.Errorf("unable to determine module name")
	}
	astPackages, err := parser.ParseDir(fset, ".", filter, parser.ParseComments)
	if err != err {
		return err
	}
	tmpl, err := template.New("new").Funcs(templateFuncs(version)).Parse(Markdown)
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
		if err = tmpl.Execute(os.Stdout, pkg); err != nil {
			log.Error("Error from tmpl.Execute")
			return err
		}
	}
	return nil
}
