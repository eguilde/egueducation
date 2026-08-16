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
		r := scalar(t.X, structs, visiting); if kind, ok := r["type"].(string); ok { r["type"] = []string{kind, "null"} }; return r
	case *ast.ArrayType:
		return map[string]any{"type":"array", "items":scalar(t.Elt, structs, visiting)}
	case *ast.MapType:
		return map[string]any{"type":"object", "additionalProperties":scalar(t.Value, structs, visiting), "x-free-form-property":true}
	case *ast.StructType:
		return objectSchema(t, structs, visiting)
	case *ast.Ident:
		if strings.Contains(t.Name, "bool") { return map[string]any{"type":"boolean"} }
		if strings.Contains(t.Name, "int") || strings.Contains(t.Name, "float") { return map[string]any{"type":"integer"} }
		if st, ok := structs[t.Name]; ok && !visiting[t.Name] { return objectSchema(st, structs, visiting) }
		return map[string]any{"type":"string"}
	default: return map[string]any{"type":"string"}
	}
}
func objectSchema(st *ast.StructType, structs map[string]*ast.StructType, visiting map[string]bool) map[string]any {
	props:=map[string]any{}
	for _, field:=range st.Fields.List {
		if len(field.Names)==0 { // embedded local DTO: JSON embeds its exported fields.
			if ident, ok:=field.Type.(*ast.Ident); ok { if embedded, ok:=structs[ident.Name]; ok && !visiting[ident.Name] { for key, value:=range objectSchema(embedded, structs, map[string]bool{ident.Name:true})["properties"].(map[string]any) { props[key]=value } } }
			continue
		}
		if field.Tag==nil { continue }
		tag:=reflect.StructTag(strings.Trim(field.Tag.Value,"`")).Get("json"); name:=strings.Split(tag,",")[0]; if name=="" || name=="-" {continue}
		props[name]=scalar(field.Type, structs, visiting)
	}
	return map[string]any{"type":"object","additionalProperties":false,"properties":props}
}
func main() {
	args:=os.Args[1:]; if len(args)>0 && args[0]=="--" { args=args[1:] }; if len(args) != 2 { panic("usage: go-schema-helper <file> <struct>") }
	f, err := parser.ParseFile(token.NewFileSet(), args[0], nil, parser.ParseComments); if err != nil { panic(err) }
	structs:=map[string]*ast.StructType{}; for _, d:=range f.Decls { if gd,ok:=d.(*ast.GenDecl); ok { for _,s:=range gd.Specs { if ts,ok:=s.(*ast.TypeSpec);ok {if st,ok:=ts.Type.(*ast.StructType);ok {structs[ts.Name.Name]=st}} } } }
	if st,ok:=structs[args[1]];ok { result:=objectSchema(st,structs,map[string]bool{args[1]:true}); result["x-source-dto"]=fmt.Sprintf("%s|%s",args[0],args[1]); _=json.NewEncoder(os.Stdout).Encode(result); return }
	panic("struct not found")
}
