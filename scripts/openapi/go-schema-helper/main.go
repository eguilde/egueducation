package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
)

func scalar(e ast.Expr, structs map[string]*ast.StructType, visiting map[string]bool) map[string]any {
	switch t := e.(type) {
	case *ast.StarExpr:
		r := scalar(t.X, structs, visiting)
		if kind, ok := r["type"].(string); ok {
			r["type"] = []string{kind, "null"}
		}
		return r
	case *ast.ArrayType:
		return map[string]any{"type": "array", "items": scalar(t.Elt, structs, visiting)}
	case *ast.MapType:
		return map[string]any{"type": "object", "additionalProperties": scalar(t.Value, structs, visiting), "x-free-form-property": true}
	case *ast.SelectorExpr:
		// encoding/json writes json.RawMessage as the JSON value it contains, not
		// as a base64/string field. Admission snapshots are validated as objects
		// by their handlers and database constraints, so preserve that wire shape.
		if pkg, ok := t.X.(*ast.Ident); ok && pkg.Name == "json" && t.Sel.Name == "RawMessage" {
			return map[string]any{"type": "object", "additionalProperties": true, "x-free-form-property": true}
		}
		return map[string]any{"type": "string"}
	case *ast.StructType:
		return objectSchema(t, structs, visiting)
	case *ast.InterfaceType:
		return map[string]any{"x-free-form-property": true}
	case *ast.Ident:
		switch t.Name {
		case "any":
			// JSON metadata may contain numbers, booleans, arrays and objects.
			// An unconstrained value schema represents Go's any, not a string.
			return map[string]any{"x-free-form-property": true}
		case "bool":
			return map[string]any{"type": "boolean"}
		case "int8", "int16", "int32", "uint8", "uint16", "uint32":
			return map[string]any{"type": "integer", "format": "int32"}
		case "int64", "uint64":
			return map[string]any{"type": "integer", "format": "int64"}
		case "int", "uint":
			return map[string]any{"type": "integer"}
		case "float32":
			return map[string]any{"type": "number", "format": "float"}
		case "float64":
			return map[string]any{"type": "number", "format": "double"}
		}
		if st, ok := structs[t.Name]; ok && !visiting[t.Name] {
			return objectSchema(st, structs, visiting)
		}
		return map[string]any{"type": "string"}
	default:
		return map[string]any{"type": "string"}
	}
}
func objectSchema(st *ast.StructType, structs map[string]*ast.StructType, visiting map[string]bool) map[string]any {
	props := map[string]any{}
	required := make([]string, 0)
	for _, field := range st.Fields.List {
		if len(field.Names) == 0 { // embedded local DTO: JSON embeds its exported fields.
			if ident, ok := field.Type.(*ast.Ident); ok {
				if embedded, ok := structs[ident.Name]; ok && !visiting[ident.Name] {
					embeddedSchema := objectSchema(embedded, structs, map[string]bool{ident.Name: true})
					for key, value := range embeddedSchema["properties"].(map[string]any) {
						props[key] = value
					}
					if embeddedRequired, ok := embeddedSchema["required"].([]string); ok {
						required = append(required, embeddedRequired...)
					}
				}
			}
			continue
		}
		if field.Tag == nil {
			continue
		}
		tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`")).Get("json")
		parts := strings.Split(tag, ",")
		name := parts[0]
		if name == "" || name == "-" {
			continue
		}
		props[name] = scalar(field.Type, structs, visiting)
		omitEmpty := false
		for _, option := range parts[1:] {
			if option == "omitempty" {
				omitEmpty = true
				break
			}
		}
		if !omitEmpty {
			required = append(required, name)
		}
	}
	result := map[string]any{"type": "object", "additionalProperties": false, "properties": props}
	if len(required) > 0 {
		result["required"] = required
	}
	return result
}
func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) != 2 {
		panic("usage: go-schema-helper <file> <struct>")
	}
	f, err := parser.ParseFile(token.NewFileSet(), args[0], nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	structs := map[string]*ast.StructType{}
	for _, d := range f.Decls {
		if gd, ok := d.(*ast.GenDecl); ok {
			for _, s := range gd.Specs {
				if ts, ok := s.(*ast.TypeSpec); ok {
					if st, ok := ts.Type.(*ast.StructType); ok {
						structs[ts.Name.Name] = st
					}
				}
			}
		}
	}
	if st, ok := structs[args[1]]; ok {
		result := objectSchema(st, structs, map[string]bool{args[1]: true})
		result["x-source-dto"] = fmt.Sprintf("%s|%s", args[0], args[1])
		_ = json.NewEncoder(os.Stdout).Encode(result)
		return
	}
	panic("struct not found")
}
