package pkg

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
)

func variableType(variable ast.Expr) string {
	typeName := ""
	switch t := variable.(type) {
	case *ast.ArrayType:
		typeName = fmt.Sprintf("[]%s", variableType(t.Elt))
	case *ast.Ident:
		typeName = t.Name
	case *ast.SelectorExpr:
		typeName = fmt.Sprintf("%s.%s", t.X, t.Sel)
	case *ast.StarExpr:
		typeName = fmt.Sprintf("*%s", variableType(t.X))
	default:
		log.WithFields(log.Fields{
			"variable.(type))": fmt.Sprintf("%#v", variable)},
		).Fatalf("unknown variable type")
	}
	return typeName
}

func typeField(field *ast.Field) string {
	line := ""
	switch t := field.Type.(type) {
	case *ast.FuncType:
		params := []string{}
		for _, param := range t.Params.List {
			params = append(params, variableType(param.Type))
		}
		results := ""
		if t.Results != nil {
			if len(t.Results.List) == 1 {
				results = variableType(t.Results.List[0].Type)
			} else {
				r := []string{}
				for _, param := range t.Results.List {
					r = append(r, variableType(param.Type))
				}
				results = fmt.Sprintf("(%s)", strings.Join(r, ", "))
			}
		}
		line = fmt.Sprintf("    func %s(%s) %s", field.Names[0], strings.Join(params, ", "), results)
	default:
		line = fmt.Sprintf("    %s %s", field.Names[0], variableType(field.Type))
	}
	return line
}

func funcElem(funcObj doc.Func) string {
	params := []string{}
	for _, paramList := range funcObj.Decl.Type.Params.List {
		for _, param := range paramList.Names {
			params = append(params, param.Name)
		}
		last := len(params) - 1
		params[last] = params[last] + " " + variableType(paramList.Type)
	}
	results := ""
	if funcObj.Decl.Type.Results != nil {
		if len(funcObj.Decl.Type.Results.List) == 1 {
			results = " " + variableType(funcObj.Decl.Type.Results.List[0].Type)
		} else {
			r := []string{}
			for _, param := range funcObj.Decl.Type.Results.List {
				r = append(r, variableType(param.Type))
			}
			results = fmt.Sprintf(" (%s)", strings.Join(r, ", "))
		}
	}
	return fmt.Sprintf("func %s(%s)%s", funcObj.Name, strings.Join(params, ", "), results)
}

func typeElem(typeObj doc.Type) string {
	if len(typeObj.Decl.Specs) == 0 {
		return ""
	}
	lines := []string{}
	for _, spec := range typeObj.Decl.Specs {
		switch t := spec.(*ast.TypeSpec).Type.(type) {
		case *ast.InterfaceType:
			for _, field := range t.Methods.List {
				lines = append(lines, typeField(field))
			}
		case *ast.StructType:
			for _, field := range t.Fields.List {
				lines = append(lines, typeField(field))
			}
		default:
			log.WithFields(log.Fields{
				"spec.(*ast.TypeSpec).Type": fmt.Sprintf("%#v", spec.(*ast.TypeSpec).Type)},
			).Fatalf("unknown parameter type %#v", t)
		}
	}
	fields := ""
	if len(lines) > 0 {
		fields = "\n" + strings.Join(lines, "\n")
	}
	return fmt.Sprintf("type %s%s", typeObj.Name, fields)
}

func filter(info fs.FileInfo) bool {
	if strings.HasSuffix(info.Name(), "_test.go") {
		return false
	}
	if strings.HasSuffix(info.Name(), ".go") {
		return true
	}
	return false
}

func Run() error {
	fset := token.NewFileSet()
	modName, err := getPackageName(".")
	if err != nil {
		return fmt.Errorf("unable to determine module name")
	}
	astPackages, err := parser.ParseDir(fset, ".", filter, parser.ParseComments)
	if err != err {
		return err
	}
	tmpl, err := template.New("new").Funcs(template.FuncMap{
		"trim":     strings.TrimSpace,
		"funcElem": funcElem,
		"typeElem": typeElem,
	}).Parse(`
# {{ .Name }}

## <a name="pkg-doc">Overview</a>

{{ trim .Doc}}

Imports: {{ len .Imports }}

## Index
{{ range $val := .Funcs }}
{{ funcElem $val }}{{- end }}
{{ range $val := .Types }}
{{ typeElem $val }}{{- end }}
`)
	if err != err {
		log.Error("Error from tmpl.New")
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
