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

func variableType(variable ast.Expr, depth int) string {
	typeName := ""
	switch t := variable.(type) {
	case *ast.ArrayType:
		typeName = fmt.Sprintf("[]%s", variableType(t.Elt, depth))
	case *ast.Ellipsis:
		typeName = fmt.Sprintf("...%s", variableType(t.Elt, depth))
	case *ast.FuncType:
		params := []string{}
		for _, paramList := range t.Params.List {
			if len(paramList.Names) > 0 {
				for _, param := range paramList.Names {
					params = append(params, param.Name)
				}
				if paramList.Type != nil {
					last := len(params) - 1
					params[last] = params[last] + " " + variableType(paramList.Type, 0)
				}
			} else if paramList.Type != nil {
				params = []string{variableType(paramList.Type, 0)}
			}
		}
		results := ""
		if t.Results != nil {
			if len(t.Results.List) == 1 {
				results = " " + variableType(t.Results.List[0].Type, 0)
			} else {
				r := []string{}
				for _, param := range t.Results.List {
					r = append(r, variableType(param.Type, 0))
				}
				results = fmt.Sprintf(" (%s)", strings.Join(r, ", "))
			}
		}
		typeName = fmt.Sprintf("func(%s)%s", strings.Join(params, ", "), results)
	case *ast.Ident:
		typeName = t.Name
	case *ast.InterfaceType:
		typeName = "interface{}"
	case *ast.MapType:
		typeName = fmt.Sprintf("map[%s]%s", variableType(t.Key, depth), variableType(t.Value, depth))
	case *ast.SelectorExpr:
		typeName = fmt.Sprintf("%s.%s", t.X, t.Sel)
	case *ast.StarExpr:
		typeName = fmt.Sprintf("*%s", variableType(t.X, depth))
	case *ast.StructType:
		lines := []string{}
		for _, field := range t.Fields.List {
			lines = append(lines, typeField(field, depth+1))
		}
		typeName = fmt.Sprintf("struct\n%s", strings.Join(lines, "\n"))
	default:
		log.WithFields(log.Fields{
			"variable.(type)": fmt.Sprintf("%#v", variable)},
		).Panicf("unknown variable type")
	}
	return typeName
}

func typeField(field *ast.Field, depth int) string {
	line := ""
	prefix := "    "
	for i := 0; i < depth; i++ {
		prefix = prefix + "    "
	}
	switch t := field.Type.(type) {
	case *ast.FuncType:
		params := []string{}
		for _, param := range t.Params.List {
			params = append(params, variableType(param.Type, depth))
		}
		results := ""
		if t.Results != nil {
			if len(t.Results.List) == 1 {
				results = " " + variableType(t.Results.List[0].Type, depth)
			} else {
				r := []string{}
				for _, param := range t.Results.List {
					r = append(r, variableType(param.Type, depth))
				}
				results = fmt.Sprintf(" (%s)", strings.Join(r, ", "))
			}
		}
		line = fmt.Sprintf("%s- func %s(%s)%s", prefix, field.Names[0], strings.Join(params, ", "), results)
	default:
		line = fmt.Sprintf("%s- %s %s", prefix, field.Names[0], variableType(field.Type, depth))
	}
	return line
}

func funcElem(funcObj doc.Func) string {
	receiver := ""
	if funcObj.Recv != "" {
		recv := funcObj.Decl.Recv.List[0]
		receiver = fmt.Sprintf("(%s %s) ", recv.Names[0], funcObj.Recv)
	}
	params := []string{}
	for _, paramList := range funcObj.Decl.Type.Params.List {
		for _, param := range paramList.Names {
			params = append(params, param.Name)
		}
		last := len(params) - 1
		params[last] = params[last] + " " + variableType(paramList.Type, 0)
	}
	results := ""
	if funcObj.Decl.Type.Results != nil {
		if len(funcObj.Decl.Type.Results.List) == 1 {
			results = " " + variableType(funcObj.Decl.Type.Results.List[0].Type, 0)
		} else {
			r := []string{}
			for _, param := range funcObj.Decl.Type.Results.List {
				r = append(r, variableType(param.Type, 0))
			}
			results = fmt.Sprintf(" (%s)", strings.Join(r, ", "))
		}
	}
	return fmt.Sprintf("- func %s%s(%s)%s", receiver, funcObj.Name, strings.Join(params, ", "), results)
}

func funcSection(funcObj doc.Func) string {
	receiver := ""
	if funcObj.Recv != "" {
		recv := funcObj.Decl.Recv.List[0]
		receiver = fmt.Sprintf("(%s %s) ", recv.Names[0], funcObj.Recv)
	}
	params := []string{}
	for _, paramList := range funcObj.Decl.Type.Params.List {
		for _, param := range paramList.Names {
			params = append(params, param.Name)
		}
		last := len(params) - 1
		params[last] = params[last] + " " + variableType(paramList.Type, 0)
	}
	results := ""
	if funcObj.Decl.Type.Results != nil {
		if len(funcObj.Decl.Type.Results.List) == 1 {
			results = " " + variableType(funcObj.Decl.Type.Results.List[0].Type, 0)
		} else {
			r := []string{}
			for _, param := range funcObj.Decl.Type.Results.List {
				r = append(r, variableType(param.Type, 0))
			}
			results = fmt.Sprintf(" (%s)", strings.Join(r, ", "))
		}
	}
	docMd := ""
	if funcObj.Doc != "" {
		docMd = "\n\n" + funcObj.Doc
	}
	return fmt.Sprintf(
		"### func %s%s(%s)%s%s", receiver, funcObj.Name, strings.Join(params, ", "), results, docMd)
}

func typeElem(typeObj doc.Type) string {
	if len(typeObj.Decl.Specs) == 0 {
		return ""
	}
	lines := []string{}
	for _, spec := range typeObj.Decl.Specs {
		switch t := spec.(*ast.TypeSpec).Type.(type) {
		case *ast.FuncType, *ast.Ident:
			typeDesc := fmt.Sprintf("- type %s %s", typeObj.Name, variableType(t, 0))
			return typeDesc
		case *ast.InterfaceType:
			for _, field := range t.Methods.List {
				lines = append(lines, typeField(field, 0))
			}
		case *ast.StructType:
			for _, field := range t.Fields.List {
				lines = append(lines, typeField(field, 0))
			}
			for _, funcObj := range typeObj.Funcs {
				lines = append(lines, "    "+funcElem(*funcObj))
			}
			for _, funcObj := range typeObj.Methods {
				lines = append(lines, "    "+funcElem(*funcObj))
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
	return fmt.Sprintf("- type %s%s", typeObj.Name, fields)
}

func typeSection(typeObj doc.Type) string {
	if len(typeObj.Decl.Specs) == 0 {
		return ""
	}
	lines := []string{}
	docMd := ""
	typeName := ""
	for _, spec := range typeObj.Decl.Specs {
		docMd = ""
		switch t := spec.(*ast.TypeSpec).Type.(type) {
		case *ast.FuncType, *ast.Ident:
			typeDesc := fmt.Sprintf("### type %s %s%s", typeObj.Name, variableType(t, 0), docMd)
			return typeDesc
		case *ast.InterfaceType:
			typeName = "interface"
		case *ast.StructType:
			typeName = "struct"
			if len(t.Fields.List) > 0 {
				lines = append(lines, "")
			}
			for _, field := range t.Fields.List {
				line := strings.TrimSpace(typeField(field, 0))
				if field.Tag != nil {
					line = line + " " + field.Tag.Value
				}
				lines = append(lines, line)
			}
			if len(t.Fields.List) > 0 {
				lines = append(lines, "")
			}
			for _, funcObj := range typeObj.Funcs {
				lines = append(lines, funcSection(*funcObj))
			}
			if len(typeObj.Methods) > 0 {
				lines = append(lines, "")
			}
			for _, funcObj := range typeObj.Methods {
				lines = append(lines, funcSection(*funcObj))
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
	if typeObj.Doc != "" {
		docMd = "\n\n" + strings.TrimSpace(typeObj.Doc)
	}
	return fmt.Sprintf("### type %s %s%s%s\n", typeObj.Name, typeName, docMd, fields)
}

func varElem(varObj doc.Value) string {
	lines := []string{}
	for _, spec := range varObj.Decl.Specs {
		varItem := spec.(*ast.ValueSpec)
		params := []string{}
		for _, param := range varItem.Names {
			params = append(params, param.Name)
		}
		if varItem.Type != nil {
			last := len(params) - 1
			params[last] = params[last] + " " + variableType(varItem.Type, 0)
		}
		lines = append(lines, "- "+strings.Join(params, ", "))
	}
	return strings.Join(lines, "\n")
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
		"trim":        strings.TrimSpace,
		"funcElem":    funcElem,
		"funcSection": funcSection,
		"typeElem":    typeElem,
		"typeSection": typeSection,
		"varElem":     varElem,
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

## Examples
{{ if .Examples }}
{{range $val := .Examples }}
- {{ $val }}
{{- end}}
{{- else }}
This section is empty.
{{- end}}

## Constants
{{ if .Consts }}
{{ range $val := .Consts }}
- {{ $val.Doc }}
{{- end}}
{{- else }}
This section is empty.
{{- end}}

## Variables
{{ if .Vars }}{{ range $val := .Vars }}
{{ varElem $val }}{{- end}}
{{- else }}
This section is empty.
{{- end}}

{{ if .Funcs }}## Functions
{{ range $val := .Funcs }}
{{ funcSection $val }}{{- end }}
{{- end }}
{{ if .Types }}## Types
{{ range $val := .Types }}
{{ typeSection $val }}{{- end }}
{{- end }}
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
		if err = tmpl.Execute(os.Stdout, *pkg); err != nil {
			log.Error("Error from tmpl.Execute")
			return err
		}
	}
	return nil
}
