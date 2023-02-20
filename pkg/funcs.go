package pkg

import (
	"fmt"
	"go/ast"
	"go/doc"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
)

var (
	basePrefix      = "    "
	exportedType, _ = regexp.Compile("^[A-Z]")
)

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
	case "bool", "byte", "char", "error", "float", "float32", "float64", "int", "int32", "int64", "string":
		return text
	}
	if !strings.Contains(text, ".") {
		if exportedType.MatchString(text) {
			return fmt.Sprintf(`<a href="#%s">%s</a>`, intoLink("type "+text), text)
		}
		log.Warningf("Internal type: %s", text)
		return text
	}
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

func typeField(field *ast.Field, depth int, hyphen bool, imports map[string]string) varTypeOutput {
	prefix := ""
	for i := 0; i <= depth; i++ {
		prefix = prefix + basePrefix
	}
	if hyphen {
		prefix = prefix + "- "
	}
	switch t := field.Type.(type) {
	case *ast.FuncType:
		fparams := funcParams(t.Params, imports)
		freturns := funcReturns(t.Results, imports)
		msg := fmt.Sprintf("%sfunc %s(%%s)%%s", prefix, field.Names[0])
		return sprintf(msg, fparams, freturns)
	default:
		vto := variableType(field.Type, depth, hyphen, imports)
		plainIdx := strings.LastIndex(vto.plainText, "\n}")
		mdIdx := strings.LastIndex(vto.markdown, "\n}")
		if plainIdx != -1 {
			vto.plainText = vto.plainText[:plainIdx+1] + prefix + vto.plainText[plainIdx+1:]
			vto.markdown = vto.markdown[:mdIdx+1] + prefix + vto.markdown[mdIdx+1:]
		}
		msg := fmt.Sprintf("%s%s %%s", prefix, field.Names[0])
		return sprintf(msg, vto)
	}
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
func funcParams(fields *ast.FieldList, imports map[string]string) varTypeOutput {
	if fields == nil {
		return sprintf("")
	}
	varTypes := []varTypeOutput{}
	for _, paramList := range fields.List {
		vto := variableType(paramList.Type, 0, false, imports)
		if len(paramList.Names) == 0 {
			varTypes = append(varTypes, vto)
			continue
		}
		nameList := []string{}
		for _, param := range paramList.Names {
			nameList = append(nameList, param.Name)
		}
		names := strings.Join(nameList, ", ")
		vto = sprintf(names+" %s", vto)
		varTypes = append(varTypes, vto)
	}
	return join(varTypes, ", ")
}

// funcReturns combines function return values into string.
// If you start from "funcObj doc.Func", you will get ast.Field from
// "funcObj.Decl.Type.Results"
func funcReturns(fields *ast.FieldList, imports map[string]string) varTypeOutput {
	switch {
	case fields == nil:
		return sprintf("")
	case len(fields.List) == 1:
		vto := variableType(fields.List[0].Type, 0, false, imports)
		return sprintf(" %s", vto)
	default:
		varTypes := []varTypeOutput{}
		for _, param := range fields.List {
			varTypes = append(varTypes, variableType(param.Type, 0, false, imports))
		}
		return sprintf(" (%s)", join(varTypes, ", "))
	}
}

func funcHeading(funcObj doc.Func) string {
	return fmt.Sprintf("func %s%s", funcReceiver(funcObj), funcObj.Name)
}

func funcElem(funcObj doc.Func) string {
	text := fmt.Sprintf(
		"func %s%s(%s)%s", funcReceiver(funcObj), funcObj.Name,
		funcParams(funcObj.Decl.Type.Params, nil).plainText,
		funcReturns(funcObj.Decl.Type.Results, nil).plainText,
	)
	link := intoLink(fmt.Sprintf("func %s%s", funcReceiver(funcObj), funcObj.Name))
	return fmt.Sprintf("- [%s](#%s)", text, link)
}

func funcSection(imports map[string]string) func(doc.Func) string {
	return func(funcObj doc.Func) string {
		return fmt.Sprintf(
			"func %s%s(%s)%s", funcReceiver(funcObj), funcObj.Name,
			funcParams(funcObj.Decl.Type.Params, imports).markdown,
			funcReturns(funcObj.Decl.Type.Results, imports).markdown,
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
				typeDesc := fmt.Sprintf("<pre>\ntype %s %s\n</pre>\n", typeObj.Name, variableType(t, 0, false, imports).markdown)
				return typeDesc
			case *ast.InterfaceType:
				typeName = "interface"
				for _, field := range t.Methods.List {
					lines = append(lines, typeField(field, 0, false, imports).markdown)
				}
				if len(t.Methods.List) > 0 {
					lines = append(lines, "")
				}
			case *ast.StructType:
				typeName = "struct"
				maxLength := 0
				structLines := []string{}
				diffLen := []int{}
				for _, field := range t.Fields.List {
					info := typeField(field, 0, false, imports)
					plainLen := len(strings.Split(info.plainText, "\n")[0])
					mdLen := len(strings.Split(info.markdown, "\n")[0])
					if plainLen > maxLength {
						maxLength = plainLen
					}
					diffLen = append(diffLen, mdLen-plainLen)
					structLines = append(structLines, info.markdown)
				}
				for idx, field := range t.Fields.List {
					line := structLines[idx]
					if field.Tag != nil {
						msg := fmt.Sprintf("%%-%ds %%s", maxLength+diffLen[idx])
						line = fmt.Sprintf(msg, line, field.Tag.Value)
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
				paramType = " " + variableType(varItem.Type, 0, false, imports).plainText
			}
			paramName := ""
			if len(varItem.Names) > 0 {
				paramName = " " + varItem.Names[0].Name
			}
			paramValue := ""
			switch len(varItem.Values) {
			case 0:
			case 1:
				value := variableType(varItem.Values[0], 0, false, imports).plainText
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
					v := variableType(value, 0, false, imports).plainText
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
