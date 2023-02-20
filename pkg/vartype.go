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

func varTypeJoin(elems []varTypeOutput, sep string) varTypeOutput {
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

func plainTextVarType(value string) varTypeOutput {
	return varTypeOutput{
		plainText: value,
		markdown:  value,
	}
}

func (vto *varTypeOutput) replace(old, new string, n int) {
	vto.plainText = strings.Replace(vto.plainText, old, new, n)
	vto.markdown = strings.Replace(vto.markdown, old, new, n)
}

func (vto *varTypeOutput) prefix(prefixText string, inLinkText bool) varTypeOutput {
	vto.plainText = prefixText + vto.plainText
	if inLinkText && strings.HasPrefix(vto.markdown, "<a href=") {
		vto.markdown = strings.Replace(vto.markdown, `">`, `">`+prefixText, 1)
	} else {
		vto.markdown = prefixText + vto.markdown
	}
	return *vto
}

func variableType(variable ast.Expr, depth int, hyphen bool, imports map[string]string) varTypeOutput {
	switch t := variable.(type) {
	case nil:
		return plainTextVarType("nil")
	case *ast.ArrayType:
		varType := variableType(t.Elt, depth, hyphen, imports)
		return varType.prefix("[]", true)
	case *ast.BasicLit:
		if t.Value != "" {
			return varTypeOutput{
				plainText: t.Value,
				markdown:  fmt.Sprintf(`<a href="#%s">%s</a>`, intoLink("type "+t.Value), t.Value),
			}
		}
		switch t.Kind {
		case token.INT:
			return plainTextVarType("int")
		case token.FLOAT:
			return plainTextVarType("float")
		case token.IMAG:
			return plainTextVarType("imag")
		case token.CHAR:
			return plainTextVarType("char")
		case token.STRING:
			return plainTextVarType("string")
		default:
			log.WithField("t.Kind", t.Kind).Panic("unknown token kind")
		}
	case *ast.CallExpr:
		funcName := variableType(t.Fun, depth, hyphen, imports).plainText
		varTypes := []varTypeOutput{}
		for _, arg := range t.Args {
			varTypes = append(varTypes, variableType(arg, depth, hyphen, imports))
		}
		vto := varTypeJoin(varTypes, ", ")
		msg := funcName + "(%s)"
		return varTypeOutput{
			plainText: fmt.Sprintf(msg, vto.plainText),
			markdown:  fmt.Sprintf(msg, vto.markdown),
		}
	case *ast.CompositeLit:
		eltsType := variableType(t.Type, depth, hyphen, imports)
		varTypes := []varTypeOutput{}
		for _, elt := range t.Elts {
			varTypes = append(varTypes, variableType(elt, depth, hyphen, imports))
		}
		switch subType := t.Type.(type) {
		case *ast.ArrayType, *ast.MapType, *ast.SelectorExpr:
			vto := varTypeJoin(varTypes, ",\n")
			vto.replace("\n", "\n"+basePrefix, -1)
			msg := fmt.Sprintf("%s{\n%s%%s,\n}", eltsType.plainText, basePrefix)
			return varTypeOutput{
				plainText: fmt.Sprintf(msg, vto.plainText),
				markdown:  fmt.Sprintf(msg, vto.markdown),
			}
		case nil:
			return varTypeJoin(varTypes, ",\n")
		default:
			log.Panicf("Unknown CompositeLit: %#v", subType)
		}
	case *ast.Ellipsis:
		varType := variableType(t.Elt, depth, hyphen, imports)
		return varType.prefix("...", false)
	case *ast.FuncType:
		vtoParams := funcParams(t.Params, imports)
		vtoReturns := funcReturns(t.Results, imports)
		msg := "func(%s)%s"
		return varTypeOutput{
			plainText: fmt.Sprintf(msg, vtoParams.plainText, vtoReturns.plainText),
			markdown:  fmt.Sprintf(msg, vtoParams.markdown, vtoReturns.markdown),
		}
	case *ast.Ident:
		switch t.Name {
		case "bool", "byte", "char", "error", "float", "float32", "float64", "int", "int32", "int64", "string":
			return plainTextVarType(t.Name)
		}
		if exportedType.MatchString(t.Name) {
			return varTypeOutput{
				plainText: t.Name,
				markdown:  fmt.Sprintf(`<a href="#%s">%s</a>`, intoLink("type "+t.Name), t.Name),
			}
		}
		log.Warningf("Internal type: %s", t.Name)
		return plainTextVarType(t.Name)
	case *ast.InterfaceType:
		return plainTextVarType("interface{}")
	case *ast.KeyValueExpr:
		keyType := variableType(t.Key, depth, hyphen, imports)
		valueType := variableType(t.Value, depth, hyphen, imports)
		switch t.Value.(type) {
		case *ast.CompositeLit:
			switch t.Key.(type) {
			case *ast.BasicLit:
				msg := "%s: {\n%s%s,\n}"
				valueType.replace("\n", "\n"+basePrefix, -1)
				return varTypeOutput{
					plainText: fmt.Sprintf(msg, keyType.plainText, basePrefix, valueType.plainText),
					markdown:  fmt.Sprintf(msg, keyType.markdown, basePrefix, valueType.markdown),
				}
			}
		}
		return varTypeOutput{
			plainText: keyType.plainText + ": " + valueType.plainText,
			markdown:  keyType.markdown + ": " + valueType.markdown,
		}
	case *ast.MapType:
		keyType := variableType(t.Key, depth, hyphen, imports)
		valueType := variableType(t.Value, depth, hyphen, imports)
		msg := "map[%s]%s"
		return varTypeOutput{
			plainText: fmt.Sprintf(msg, keyType.plainText, valueType.plainText),
			markdown:  fmt.Sprintf(msg, keyType.markdown, valueType.markdown),
		}
	case *ast.SelectorExpr:
		msg := fmt.Sprintf("%s.%s", t.X, t.Sel)
		return varTypeOutput{plainText: msg, markdown: intoImportLink(msg, imports)}
	case *ast.StarExpr:
		varType := variableType(t.X, depth, hyphen, imports)
		return varType.prefix("*", true)
	case *ast.StructType:
		varTypes := []varTypeOutput{}
		for _, field := range t.Fields.List {
			varTypes = append(varTypes, typeField(field, depth+1, hyphen, imports))
		}
		vto := varTypeJoin(varTypes, "\n")
		if hyphen {
			return vto.prefix("struct\n", false)
		}
		msg := "struct {\n%s\n}"
		return varTypeOutput{
			plainText: fmt.Sprintf(msg, vto.plainText),
			markdown:  fmt.Sprintf(msg, vto.markdown),
		}
	case *ast.UnaryExpr:
		switch t.Op {
		case token.AND:
			vto := variableType(t.X, depth, hyphen, imports)
			return vto.prefix("&", true)
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
