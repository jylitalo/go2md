package pkg

import (
	"fmt"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
)

func Run() error {
	fset := token.NewFileSet()
	modName, err := getPackageName(".")
	if err != nil {
		return fmt.Errorf("unable to determine module name")
	}
	log.Info("modName=" + modName)
	astPackages, err := parser.ParseDir(fset, ".", nil, parser.ParseComments)
	if err != err {
		return err
	}
	tmpl, err := template.New("new").Funcs(template.FuncMap{
		"trim": strings.TrimSpace,
	}).Parse(`
# {{ .Name }}

## <a name="pkg-doc">Overview</a>

{{ trim .Doc}}

Imports: {{ len .Imports }}
`)
	if err != err {
		return err
	}
	for _, astPkg := range astPackages {
		pkg := doc.New(astPkg, ".", 0)
		// log.WithFields(log.Fields{"pkg": fmt.Sprintf("%#v", pkg)}).Info("output from doc.New")
		if strings.HasSuffix(modName, "/"+pkg.Name) {
			pkg.Name = modName
		}
		if err = tmpl.Execute(os.Stdout, *pkg); err != err {
			log.Error("Error from tmpl.Execute")
			return err
		}
	}
	return nil
}
