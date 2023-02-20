package pkg

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	log "github.com/sirupsen/logrus"
)

type varTypeOutput struct {
	plainText string
	markdown  string
}

func join(elems []varTypeOutput, sep string) varTypeOutput {
	plains := []string{}
	markdowns := []string{}
	for _, item := range elems {
		plains = append(plains, item.plainText)
		markdowns = append(markdowns, item.markdown)
	}
	return varTypeOutput{
		plainText: strings.Join(plains, sep),
		markdown:  strings.Join(markdowns, sep),
	}
}

func sprintf(format string, elems ...varTypeOutput) varTypeOutput {
	plainText := []any{}
	markdown := []any{}
	for _, elem := range elems {
		plainText = append(plainText, elem.plainText)
		markdown = append(markdown, elem.markdown)
	}
	return varTypeOutput{
		plainText: fmt.Sprintf(format, plainText...),
		markdown:  fmt.Sprintf(format, markdown...),
	}
}

func (vto *varTypeOutput) prefix(prefixText string) varTypeOutput {
	vto.plainText = prefixText + vto.plainText
	if strings.HasPrefix(vto.markdown, "<a href=") {
		vto.markdown = strings.Replace(vto.markdown, `">`, `">`+prefixText, 1)
	} else {
		vto.markdown = prefixText + vto.markdown
	}
	return *vto
}

func (vto *varTypeOutput) replace(old, new string, n int) {
	vto.plainText = strings.Replace(vto.plainText, old, new, n)
	vto.markdown = strings.Replace(vto.markdown, old, new, n)
}

func variableType(variable ast.Expr, depth int, hyphen bool, imports map[string]string) varTypeOutput {
	switch t := variable.(type) {
	case nil:
		return sprintf("nil")
	case *ast.ArrayType:
		varType := variableType(t.Elt, depth, hyphen, imports)
		return varType.prefix("[]")
	case *ast.BasicLit:
		if t.Value != "" {
			return varTypeOutput{
				plainText: t.Value,
				markdown:  fmt.Sprintf(`<a href="#%s">%s</a>`, intoLink("type "+t.Value), t.Value),
			}
		}
		switch t.Kind {
		case token.INT:
			return sprintf("int")
		case token.FLOAT:
			return sprintf("float")
		case token.IMAG:
			return sprintf("imag")
		case token.CHAR:
			return sprintf("char")
		case token.STRING:
			return sprintf("string")
		default:
			log.WithField("t.Kind", t.Kind).Panic("unknown token kind")
		}
	case *ast.CallExpr:
		funcName := variableType(t.Fun, depth, hyphen, imports).plainText
		varTypes := []varTypeOutput{}
		for _, arg := range t.Args {
			varTypes = append(varTypes, variableType(arg, depth, hyphen, imports))
		}
		return sprintf(funcName+"(%s)", join(varTypes, ", "))
	case *ast.CompositeLit:
		eltsType := variableType(t.Type, depth, hyphen, imports)
		varTypes := []varTypeOutput{}
		for _, elt := range t.Elts {
			varTypes = append(varTypes, variableType(elt, depth, hyphen, imports))
		}
		switch subType := t.Type.(type) {
		case *ast.ArrayType, *ast.MapType, *ast.SelectorExpr:
			vto := join(varTypes, ",\n")
			vto.replace("\n", "\n"+basePrefix, -1)
			msg := fmt.Sprintf("%s{\n%s%%s,\n}", eltsType.plainText, basePrefix)
			return sprintf(msg, vto)
		case nil:
			return join(varTypes, ",\n")
		default:
			log.Panicf("Unknown CompositeLit: %#v", subType)
		}
	case *ast.Ellipsis:
		return sprintf("...%s", variableType(t.Elt, depth, hyphen, imports))
	case *ast.FuncType:
		vtoParams := funcParams(t.Params, imports)
		vtoReturns := funcReturns(t.Results, imports)
		return sprintf("func(%s)%s", vtoParams, vtoReturns)
	case *ast.Ident:
		switch t.Name {
		case "bool", "byte", "char", "error", "float", "float32", "float64", "int", "int32", "int64", "string":
			return sprintf(t.Name)
		}
		if exportedType.MatchString(t.Name) {
			return varTypeOutput{
				plainText: t.Name,
				markdown:  fmt.Sprintf(`<a href="#%s">%s</a>`, intoLink("type "+t.Name), t.Name),
			}
		}
		log.Warningf("Internal type: %s", t.Name)
		return sprintf(t.Name)
	case *ast.InterfaceType:
		return sprintf("interface{}")
	case *ast.KeyValueExpr:
		keyType := variableType(t.Key, depth, hyphen, imports)
		valueType := variableType(t.Value, depth, hyphen, imports)
		switch t.Value.(type) {
		case *ast.CompositeLit:
			switch t.Key.(type) {
			case *ast.BasicLit:
				valueType.replace("\n", "\n"+basePrefix, -1)
				valueType = sprintf(basePrefix+"%s", valueType)
				return sprintf("%s: {\n%s,\n}", keyType, valueType)
			}
		}
		return sprintf("%s: %s", keyType, valueType)
	case *ast.MapType:
		keyType := variableType(t.Key, depth, hyphen, imports)
		valueType := variableType(t.Value, depth, hyphen, imports)
		return sprintf("map[%s]%s", keyType, valueType)
	case *ast.SelectorExpr:
		msg := fmt.Sprintf("%s.%s", t.X, t.Sel)
		return varTypeOutput{plainText: msg, markdown: intoImportLink(msg, imports)}
	case *ast.StarExpr:
		vto := variableType(t.X, depth, hyphen, imports)
		return vto.prefix("*")
	case *ast.StructType:
		varTypes := []varTypeOutput{}
		for _, field := range t.Fields.List {
			varTypes = append(varTypes, typeField(field, depth+1, hyphen, imports))
		}
		vto := join(varTypes, "\n")
		if hyphen {
			return sprintf("struct\n%s", vto)
		}
		return sprintf("struct {\n%s\n}", vto)
	case *ast.UnaryExpr:
		switch t.Op {
		case token.AND:
			vto := variableType(t.X, depth, hyphen, imports)
			return vto.prefix("&")
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
