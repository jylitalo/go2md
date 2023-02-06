package pkg

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/token"
	"path/filepath"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
)

type varTypeOutput struct {
	full string
	link string
}

func templateFuncs(version string, imports map[string]string) template.FuncMap {
	return template.FuncMap{
		"trim":        strings.TrimSpace,
		"funcElem":    funcElem,
		"funcHeading": funcHeading,
		"funcSection": funcSection(imports),
		"typeElem":    typeElem,
		"typeSection": typeSection(imports),
		"varElem":     varElem(imports),
		"version":     func() string { return version },
	}
}

func intoLink(text string) string {
	link := strings.ToLower(text)
	link = strings.ReplaceAll(link, " ", "-")
	link = strings.ReplaceAll(link, "*", "")
	link = strings.ReplaceAll(link, "(", "")
	link = strings.ReplaceAll(link, ")", "")
	return link
}

func intoImportLink(text string, imports map[string]string) string {
	if imports == nil {
		return text
	}
	switch text {
	case "bool", "char", "error", "float", "float32", "float64", "int", "int32", "int64", "string":
		return text
	}
	if strings.Contains(text, ".") {
		fields := strings.SplitN(text, ".", 2)
		if modPath, ok := imports[fields[0]]; ok {
			modFields := strings.Split(modPath, "/")
			if len(modFields) >= 3 {
				mainPath := strings.Join(strings.Split(imports["main"], "/")[0:3], "/")
				if strings.Join(modFields[0:3], "/") == mainPath {
					relPath, err := filepath.Rel(imports["main"], modPath)
					if err != nil {
						log.WithFields(log.Fields{"imports['main']": imports["main"], "modPath": modPath}).Fatal("Unable to establish relative path")
					}
					return fmt.Sprintf(`<a href="%s/README.md#%s">%s</a>`, relPath, intoLink("type "+fields[1]), text)
				}
			}
			return fmt.Sprintf(`<a href="https://pkg.go.dev/%s#%s">%s</a>`, modPath, fields[1], text)
		}
		return fmt.Sprintf(`<a href="https://pkg.go.dev/%s#%s">%s</a>`, fields[0], fields[1], text)
	}
	return fmt.Sprintf(`<a href="#%s">%s</a>`, intoLink("type "+text), text)
}

func variableType(variable ast.Expr, depth int, imports map[string]string) varTypeOutput {
	basePrefix := "    "
	switch t := variable.(type) {
	case nil:
		return varTypeOutput{full: "nil", link: "nil"}
	case *ast.ArrayType:
		varType := variableType(t.Elt, depth, imports)
		varType.full = "[]" + varType.full
		if strings.HasPrefix(varType.link, "<a href=") {
			varType.link = strings.Replace(varType.link, `">`, `">[]`, 1)
		}
		return varType
	case *ast.BasicLit:
		if t.Value != "" {
			return varTypeOutput{
				full: t.Value,
				link: fmt.Sprintf(`<a href="#%s">%s</a>`, intoLink("type "+t.Value), t.Value),
			}
		}
		switch t.Kind {
		case token.INT:
			return varTypeOutput{full: "int", link: "int"}
		case token.FLOAT:
			return varTypeOutput{full: "float", link: "float"}
		case token.IMAG:
			return varTypeOutput{full: "imag", link: "imag"}
		case token.CHAR:
			return varTypeOutput{full: "char", link: "char"}
		case token.STRING:
			return varTypeOutput{full: "string", link: "string"}
		default:
			log.WithField("t.Kind", t.Kind).Panic("unknown token kind")
		}
	case *ast.CallExpr:
		funcName := variableType(t.Fun, depth, imports).full
		fullArgs := []string{}
		linkArgs := []string{}
		for _, arg := range t.Args {
			varType := variableType(arg, depth, imports)
			fullArgs = append(fullArgs, varType.full)
			linkArgs = append(linkArgs, varType.link)
		}
		return varTypeOutput{
			full: fmt.Sprintf("%s(%s)", funcName, strings.Join(fullArgs, ", ")),
			link: fmt.Sprintf("%s(%s)", funcName, strings.Join(linkArgs, ", ")),
		}
	case *ast.CompositeLit:
		elts := []string{}
		eltsType := variableType(t.Type, depth, imports).full
		for _, elt := range t.Elts {
			elts = append(elts, variableType(elt, depth, imports).full)
		}
		switch subType := t.Type.(type) {
		case *ast.ArrayType, *ast.MapType, *ast.SelectorExpr:
			lines := strings.Split(strings.Join(elts, ",\n"), "\n")
			return varTypeOutput{full: fmt.Sprintf(
				"%s{\n%s%s,\n}", eltsType, basePrefix,
				strings.Join(lines, "\n"+basePrefix),
			)}
		case nil:
			return varTypeOutput{full: strings.Join(elts, ",\n")}
		default:
			log.Panicf("Unknown CompositeLit: %#v", subType)
		}
	case *ast.Ellipsis:
		varType := variableType(t.Elt, depth, imports)
		varType.full = "..." + varType.full
		varType.link = "..." + varType.link
		return varType
	case *ast.FuncType:
		return varTypeOutput{full: fmt.Sprintf(
			"func(%s)%s", strings.Join(funcParams(t.Params, imports), ", "),
			funcReturns(t.Results, imports),
		)}
	case *ast.Ident:
		switch t.Name {
		case "bool", "char", "error", "float", "float32", "float64", "int", "int32", "int64", "string":
			return varTypeOutput{full: t.Name, link: t.Name}
		}
		return varTypeOutput{
			full: t.Name,
			link: fmt.Sprintf(`<a href="#%s">%s</a>`, intoLink("type "+t.Name), t.Name),
		}
	case *ast.InterfaceType:
		return varTypeOutput{full: "interface{}", link: "interface{}"}
	case *ast.KeyValueExpr:
		keyType := variableType(t.Key, depth, imports)
		valueType := variableType(t.Value, depth, imports)
		switch t.Value.(type) {
		case *ast.CompositeLit:
			switch t.Key.(type) {
			case *ast.BasicLit:
				return varTypeOutput{
					full: fmt.Sprintf(
						"%s: {\n%s%s,\n}", keyType.full, basePrefix,
						strings.Join(strings.Split(valueType.full, "\n"), "\n"+basePrefix),
					),
					link: fmt.Sprintf(
						"%s: {\n%s%s,\n}", keyType.full, basePrefix,
						strings.Join(strings.Split(valueType.full, "\n"), "\n"+basePrefix),
					),
				}
			}
		}
		return varTypeOutput{
			full: keyType.full + ": " + valueType.full,
			link: keyType.link + ": " + valueType.link,
		}
	case *ast.MapType:
		keyType := variableType(t.Key, depth, imports)
		valueType := variableType(t.Value, depth, imports)
		return varTypeOutput{
			full: fmt.Sprintf("map[%s]%s", keyType.full, valueType.full),
			link: fmt.Sprintf("map[%s]%s", keyType.link, valueType.link),
		}
	case *ast.SelectorExpr:
		return varTypeOutput{
			full: fmt.Sprintf("%s.%s", t.X, t.Sel),
			link: intoImportLink(fmt.Sprintf("%s.%s", t.X, t.Sel), imports),
		}
	case *ast.StarExpr:
		varType := variableType(t.X, depth, imports)
		varType.full = "*" + varType.full
		if strings.HasPrefix(varType.link, "<a href=") {
			varType.link = strings.Replace(varType.link, `">`, `">*`, 1)
		} else {
			varType.link = "*" + varType.link
		}
		return varType
	case *ast.StructType:
		lines := []string{}
		for _, field := range t.Fields.List {
			lines = append(lines, typeField(field, depth+1, true, imports))
		}
		return varTypeOutput{
			full: fmt.Sprintf("struct\n%s", strings.Join(lines, "\n")),
			link: fmt.Sprintf("struct\n%s", strings.Join(lines, "\n")),
		}
	case *ast.UnaryExpr:
		switch t.Op {
		case token.AND:
			varType := variableType(t.X, depth, imports)
			varType.full = "&" + varType.full
			if strings.HasPrefix(varType.link, "<a href=") {
				varType.link = strings.Replace(varType.link, `">`, `">&`, 1)
			} else {
				varType.link = "&" + varType.link
			}
			return varType
		default:
			log.Panicf("unknown unary type %d for %s", t.Op, t.X)
		}
	default:
		log.WithFields(log.Fields{
			"variable.(type)": fmt.Sprintf("%#v", variable)},
		).Panicf("unknown variable type")
	}
	log.WithField("variableType", fmt.Sprintf("%#v", variable))
	return varTypeOutput{}
}

func typeField(field *ast.Field, depth int, hyphen bool, imports map[string]string) string {
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
		line = fmt.Sprintf(
			"%sfunc %s(%s)%s", prefix, field.Names[0],
			strings.Join(funcParams(t.Params, imports), ", "), funcReturns(t.Results, imports))
	default:
		line = fmt.Sprintf("%s%s %s", prefix, field.Names[0], variableType(field.Type, depth, imports).link)
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

// funcParams combines function parameters into string.
// If you start from "funcObj doc.Func", you will get ast.Field from
// "funcObj.Decl.Type.Params.List"
func funcParams(fields *ast.FieldList, imports map[string]string) []string {
	if fields == nil {
		return nil
	}
	params := []string{}
	for _, paramList := range fields.List {
		varType := variableType(paramList.Type, 0, imports)
		paramType := varType.full
		if imports != nil {
			paramType = varType.link
		}
		if len(paramList.Names) == 0 {
			params = append(params, paramType)
			continue
		}
		for _, param := range paramList.Names {
			params = append(params, param.Name)
		}
		last := len(params) - 1
		params[last] = params[last] + " " + paramType
	}
	return params
}

// funcReturns combines function return values into string.
// If you start from "funcObj doc.Func", you will get ast.Field from
// "funcObj.Decl.Type.Results"
func funcReturns(fields *ast.FieldList, imports map[string]string) string {
	switch {
	case fields == nil:
		return ""
	case len(fields.List) == 1 && imports == nil:
		return " " + variableType(fields.List[0].Type, 0, nil).full
	case len(fields.List) == 1:
		return " " + variableType(fields.List[0].Type, 0, imports).link
	case imports == nil:
		r := []string{}
		for _, param := range fields.List {
			r = append(r, variableType(param.Type, 0, imports).full)
		}
		return fmt.Sprintf(" (%s)", strings.Join(r, ", "))
	default:
		r := []string{}
		for _, param := range fields.List {
			r = append(r, variableType(param.Type, 0, imports).link)
		}
		return fmt.Sprintf(" (%s)", strings.Join(r, ", "))
	}
}

func funcHeading(funcObj doc.Func) string {
	return fmt.Sprintf("func %s%s", funcReceiver(funcObj), funcObj.Name)
}

func funcElem(funcObj doc.Func) string {
	text := fmt.Sprintf(
		"func %s%s(%s)%s", funcReceiver(funcObj), funcObj.Name,
		strings.Join(funcParams(funcObj.Decl.Type.Params, nil), ", "),
		funcReturns(funcObj.Decl.Type.Results, nil),
	)
	link := intoLink(fmt.Sprintf("func %s%s", funcReceiver(funcObj), funcObj.Name))
	return fmt.Sprintf("- [%s](#%s)", text, link)
}

func funcSection(imports map[string]string) func(doc.Func) string {
	return func(funcObj doc.Func) string {
		return fmt.Sprintf(
			"func %s%s(%s)%s", funcReceiver(funcObj), funcObj.Name,
			strings.Join(funcParams(funcObj.Decl.Type.Params, imports), ", "),
			funcReturns(funcObj.Decl.Type.Results, imports),
		)
	}
}

func typeElem(typeObj doc.Type) string {
	if len(typeObj.Decl.Specs) == 0 {
		return ""
	}
	lines := []string{}
	for _, spec := range typeObj.Decl.Specs {
		switch t := spec.(*ast.TypeSpec).Type.(type) {
		case *ast.FuncType, *ast.Ident:
			typeDesc := fmt.Sprintf("- type %s", typeObj.Name)
			return typeDesc
		case *ast.InterfaceType:
			link := intoLink("type " + typeObj.Name)
			typeDesc := fmt.Sprintf("- [type %s](#%s)", typeObj.Name, link)
			return typeDesc
		case *ast.StructType:
			funcs := append(typeObj.Funcs, typeObj.Methods...)
			for _, funcObj := range funcs {
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
	return fmt.Sprintf("- [type %s](#type-%s)%s", typeObj.Name, strings.ToLower(typeObj.Name), fields)
}

func typeSection(imports map[string]string) func(doc.Type) string {
	return func(typeObj doc.Type) string {
		if len(typeObj.Decl.Specs) == 0 {
			return ""
		}
		lines := []string{}
		typeName := ""
		for _, spec := range typeObj.Decl.Specs {
			switch t := spec.(*ast.TypeSpec).Type.(type) {
			case *ast.FuncType, *ast.Ident:
				typeDesc := fmt.Sprintf("<pre>\ntype %s %s\n</pre>\n", typeObj.Name, variableType(t, 0, imports).full)
				return typeDesc
			case *ast.InterfaceType:
				typeName = "interface"
				for _, field := range t.Methods.List {
					lines = append(lines, typeField(field, 0, false, imports))
				}
				if len(t.Methods.List) > 0 {
					lines = append(lines, "")
				}
			case *ast.StructType:
				typeName = "struct"
				for _, field := range t.Fields.List {
					line := typeField(field, 0, false, imports)
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
		return fmt.Sprintf("type %s %s {%s}", typeObj.Name, typeName, fields)
	}
}

func varElem(imports map[string]string) func(doc.Value, string) string {
	return func(varObj doc.Value, varType string) string {
		lines := []string{}
		for _, spec := range varObj.Decl.Specs {
			varItem := spec.(*ast.ValueSpec)
			paramType := ""
			if varItem.Type != nil {
				paramType = " " + variableType(varItem.Type, 0, imports).full
			}
			paramName := ""
			if len(varItem.Names) > 0 {
				paramName = " " + varItem.Names[0].Name
			}
			paramValue := ""
			switch len(varItem.Values) {
			case 0:
			case 1:
				value := variableType(varItem.Values[0], 0, imports).full
				value = strings.Trim(value, paramType)
				switch varItem.Values[0].(type) {
				case *ast.ArrayType, *ast.MapType:
					value = paramType + value
					paramType = ""
				}
				paramValue = fmt.Sprintf(" = %s", value)
			default:
				values := []string{}
				for _, value := range varItem.Values {
					v := variableType(value, 0, imports).full
					switch value.(type) {
					case *ast.ArrayType, *ast.MapType:
						v = paramType + v
						paramType = ""
					}
					values = append(values, v)
				}
				paramValue = fmt.Sprintf(" {\n%s\n}", strings.Join(values, ", "))
			}
			paramComment := ""
			if varItem.Comment != nil {
				comments := []string{}
				for _, comment := range varItem.Comment.List {
					comments = append(comments, comment.Text)
				}
				paramComment = " " + strings.Join(comments, "\n")
			}
			lines = append(lines, fmt.Sprintf("%s%s%s%s%s", varType, paramName, paramType, paramValue, paramComment))
		}
		return strings.Join(lines, "\n")
	}
}
