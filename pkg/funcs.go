package pkg

import (
	"fmt"
	"go/ast"
	"go/doc"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
)

func templateFuncs(version string) template.FuncMap {
	return template.FuncMap{
		"trim":        strings.TrimSpace,
		"funcElem":    funcElem,
		"funcHeading": funcHeading,
		"funcSection": funcSection,
		"typeElem":    typeElem,
		"typeSection": typeSection,
		"varElem":     varElem,
		"version":     func() string { return version },
	}
}

func code(text string) string {
	return "```golang\n" + text + "\n```\n"
}

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
			lines = append(lines, typeField(field, depth+1, true))
		}
		typeName = fmt.Sprintf("struct\n%s", strings.Join(lines, "\n"))
	default:
		log.WithFields(log.Fields{
			"variable.(type)": fmt.Sprintf("%#v", variable)},
		).Panicf("unknown variable type")
	}
	return typeName
}

func typeField(field *ast.Field, depth int, hyphen bool) string {
	line := ""
	prefix := "    "
	for i := 0; i < depth; i++ {
		prefix = prefix + "    "
	}
	if hyphen {
		prefix = prefix + "- "
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
		line = fmt.Sprintf("%sfunc %s(%s)%s", prefix, field.Names[0], strings.Join(params, ", "), results)
	default:
		line = fmt.Sprintf("%s%s %s", prefix, field.Names[0], variableType(field.Type, depth))
	}
	return line
}

func funcReceiver(funcObj doc.Func) string {
	receiver := ""
	if funcObj.Recv != "" {
		recv := funcObj.Decl.Recv.List[0]
		receiver = fmt.Sprintf("(%s %s) ", recv.Names[0], funcObj.Recv)
	}
	return receiver
}

func funcParams(funcObj doc.Func) []string {
	params := []string{}
	for _, paramList := range funcObj.Decl.Type.Params.List {
		for _, param := range paramList.Names {
			params = append(params, param.Name)
		}
		last := len(params) - 1
		params[last] = params[last] + " " + variableType(paramList.Type, 0)
	}
	return params
}

func funcReturns(funcObj doc.Func) string {
	switch {
	case funcObj.Decl.Type.Results == nil:
		return ""
	case len(funcObj.Decl.Type.Results.List) == 1:
		return " " + variableType(funcObj.Decl.Type.Results.List[0].Type, 0)
	default:
		r := []string{}
		for _, param := range funcObj.Decl.Type.Results.List {
			r = append(r, variableType(param.Type, 0))
		}
		return fmt.Sprintf(" (%s)", strings.Join(r, ", "))
	}
}

func funcHeading(funcObj doc.Func) string {
	return fmt.Sprintf("func %s%s", funcReceiver(funcObj), funcObj.Name)
}

func funcElem(funcObj doc.Func) string {
	return fmt.Sprintf(
		"- func %s%s(%s)%s", funcReceiver(funcObj), funcObj.Name,
		strings.Join(funcParams(funcObj), ", "), funcReturns(funcObj),
	)
}

func funcSection(funcObj doc.Func) string {
	return code(fmt.Sprintf(
		"func %s%s(%s)%s", funcReceiver(funcObj), funcObj.Name,
		strings.Join(funcParams(funcObj), ", "), funcReturns(funcObj)),
	)
}

func typeElem(typeObj doc.Type) string {
	if len(typeObj.Decl.Specs) == 0 {
		return ""
	}
	lines := []string{}
	for _, spec := range typeObj.Decl.Specs {
		switch t := spec.(*ast.TypeSpec).Type.(type) {
		case *ast.FuncType, *ast.Ident, *ast.InterfaceType:
			typeDesc := fmt.Sprintf("- type %s", typeObj.Name)
			return typeDesc
		case *ast.StructType:
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
	typeName := ""
	for _, spec := range typeObj.Decl.Specs {
		switch t := spec.(*ast.TypeSpec).Type.(type) {
		case *ast.FuncType, *ast.Ident:
			typeDesc := code(fmt.Sprintf("type %s %s", typeObj.Name, variableType(t, 0)))
			return typeDesc
		case *ast.InterfaceType:
			typeName = "interface"
			for _, field := range t.Methods.List {
				lines = append(lines, typeField(field, 0, false))
			}
		case *ast.StructType:
			typeName = "struct"
			for _, field := range t.Fields.List {
				line := typeField(field, 0, false)
				if field.Tag != nil {
					line = line + " " + field.Tag.Value
				}
				lines = append(lines, line)
			}
			if len(t.Fields.List) > 0 {
				lines = append(lines, "")
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
	return "\n" + code(fmt.Sprintf("type %s %s {%s}", typeObj.Name, typeName, fields))
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
