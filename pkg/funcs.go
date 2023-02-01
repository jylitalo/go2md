package pkg

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/token"
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

func variableType(variable ast.Expr, depth int) string {
	basePrefix := "    "
	switch t := variable.(type) {
	case nil:
		return "nil"
	case *ast.ArrayType:
		return fmt.Sprintf("[]%s", variableType(t.Elt, depth))
	case *ast.BasicLit:
		if t.Value != "" {
			return t.Value
		}
		switch t.Kind {
		case token.INT:
			return "int"
		case token.FLOAT:
			return "float"
		case token.IMAG:
			return "imag"
		case token.CHAR:
			return "char"
		case token.STRING:
			return "string"
		default:
			log.WithField("t.Kind", t.Kind).Panic("unknown token kind")
		}
	case *ast.CallExpr:
		funcName := variableType(t.Fun, depth)
		args := []string{}
		for _, arg := range t.Args {
			args = append(args, variableType(arg, depth))
		}
		return fmt.Sprintf("%s(%s)", funcName, strings.Join(args, ", "))
	case *ast.CompositeLit:
		elts := []string{}
		eltsType := variableType(t.Type, depth)
		for _, elt := range t.Elts {
			elts = append(elts, variableType(elt, depth))
		}
		switch subType := t.Type.(type) {
		case *ast.ArrayType, *ast.MapType, *ast.SelectorExpr:
			lines := strings.Split(strings.Join(elts, ",\n"), "\n")
			return fmt.Sprintf("%s{\n%s%s,\n}", eltsType, basePrefix, strings.Join(lines, "\n"+basePrefix))
		case nil:
			return strings.Join(elts, ",\n")
		default:
			log.Panicf("Unknown CompositeLit: %#v", subType)
		}
	case *ast.Ellipsis:
		return fmt.Sprintf("...%s", variableType(t.Elt, depth))
	case *ast.FuncType:
		return fmt.Sprintf(
			"func(%s)%s", strings.Join(funcParams(t.Params), ", "), funcReturns(t.Results))
	case *ast.Ident:
		return t.Name
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.KeyValueExpr:
		switch t.Value.(type) {
		case *ast.CompositeLit:
			switch t.Key.(type) {
			case *ast.BasicLit:
				lines := strings.Split(variableType(t.Value, depth), "\n")
				return fmt.Sprintf(
					"%s: {\n%s%s,\n}", variableType(t.Key, depth), basePrefix, strings.Join(lines, "\n"+basePrefix),
				)
			}
		}
		return fmt.Sprintf("%s: %s", variableType(t.Key, depth), variableType(t.Value, depth))
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", variableType(t.Key, depth), variableType(t.Value, depth))
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", t.X, t.Sel)
	case *ast.StarExpr:
		return fmt.Sprintf("*%s", variableType(t.X, depth))
	case *ast.StructType:
		lines := []string{}
		for _, field := range t.Fields.List {
			lines = append(lines, typeField(field, depth+1, true))
		}
		return fmt.Sprintf("struct\n%s", strings.Join(lines, "\n"))
	case *ast.UnaryExpr:
		switch t.Op {
		case token.AND:
			return "&" + variableType(t.X, depth)
		default:
			log.Panicf("unknown unary type %d for %s", t.Op, t.X)
		}
	default:
		log.WithFields(log.Fields{
			"variable.(type)": fmt.Sprintf("%#v", variable)},
		).Panicf("unknown variable type")
	}
	log.WithField("variableType", fmt.Sprintf("%#v", variable))
	return ""
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
		line = fmt.Sprintf(
			"%sfunc %s(%s)%s", prefix, field.Names[0],
			strings.Join(funcParams(t.Params), ", "), funcReturns(t.Results))
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

// funcParams combines function parameters into string.
// If you start from "funcObj doc.Func", you will get ast.Field from
// "funcObj.Decl.Type.Params.List"
func funcParams(fields *ast.FieldList) []string {
	if fields == nil {
		return nil
	}
	params := []string{}
	for _, paramList := range fields.List {
		paramType := variableType(paramList.Type, 0)
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
func funcReturns(fields *ast.FieldList) string {
	switch {
	case fields == nil:
		return ""
	case len(fields.List) == 1:
		return " " + variableType(fields.List[0].Type, 0)
	default:
		r := []string{}
		for _, param := range fields.List {
			r = append(r, variableType(param.Type, 0))
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
		strings.Join(funcParams(funcObj.Decl.Type.Params), ", "),
		funcReturns(funcObj.Decl.Type.Results),
	)
	link := strings.ToLower(fmt.Sprintf("func %s%s", funcReceiver(funcObj), funcObj.Name))
	link = strings.ReplaceAll(link, " ", "-")
	link = strings.ReplaceAll(link, "*", "")
	link = strings.ReplaceAll(link, "(", "")
	link = strings.ReplaceAll(link, ")", "")
	return fmt.Sprintf("- [%s](#%s)", text, link)
}

func funcSection(funcObj doc.Func) string {
	return fmt.Sprintf(
		"func %s%s(%s)%s", funcReceiver(funcObj), funcObj.Name,
		strings.Join(funcParams(funcObj.Decl.Type.Params), ", "),
		funcReturns(funcObj.Decl.Type.Results),
	)
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
			link := fmt.Sprintf("type-%s", strings.ToLower(typeObj.Name))
			typeDesc := fmt.Sprintf("- [type %s](#%s)", typeObj.Name, link)
			return typeDesc
		case *ast.StructType:
			funcs := append(typeObj.Funcs, typeObj.Methods...)
			for _, funcObj := range funcs {
				lines = append(lines, "    [%s](#func-%s)", funcElem(*funcObj), strings.ToLower(funcObj.Name))
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
			typeDesc := fmt.Sprintf("```golang\ntype %s %s\n```\n", typeObj.Name, variableType(t, 0))
			return typeDesc
		case *ast.InterfaceType:
			typeName = "interface"
			for _, field := range t.Methods.List {
				lines = append(lines, typeField(field, 0, false))
			}
			if len(t.Methods.List) > 0 {
				lines = append(lines, "")
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
	return fmt.Sprintf("type %s %s {%s}", typeObj.Name, typeName, fields)
}

func varElem(varObj doc.Value, varType string) string {
	lines := []string{}
	for _, spec := range varObj.Decl.Specs {
		varItem := spec.(*ast.ValueSpec)
		paramType := ""
		if varItem.Type != nil {
			paramType = " " + variableType(varItem.Type, 0)
		}
		paramName := ""
		if len(varItem.Names) > 0 {
			paramName = " " + varItem.Names[0].Name
		}
		paramValue := ""
		switch len(varItem.Values) {
		case 0:
		case 1:
			value := variableType(varItem.Values[0], 0)
			value = strings.Trim(value, paramType)
			if strings.HasPrefix(paramType, " []") || strings.HasPrefix(paramType, " map[") {
				value = paramType + value
				paramType = ""
			}
			paramValue = fmt.Sprintf(" = %s", value)
		default:
			values := []string{}
			for _, value := range varItem.Values {
				v := variableType(value, 0)
				if strings.HasPrefix(paramType, " []") || strings.HasPrefix(paramType, " map[") {
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
