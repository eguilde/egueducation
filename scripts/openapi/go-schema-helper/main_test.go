package main

import (
	"go/ast"
	"go/parser"
	"reflect"
	"testing"
)

func TestMetadataValueSchema(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		want       map[string]any
	}{
		{name: "any metadata", expression: "map[string]any", want: map[string]any{"x-free-form-property": true}},
		{name: "interface metadata", expression: "map[string]interface{}", want: map[string]any{"x-free-form-property": true}},
		{name: "string dictionary stays typed", expression: "map[string]string", want: map[string]any{"type": "string"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expression, err := parser.ParseExpr(tt.expression)
			if err != nil {
				t.Fatal(err)
			}
			got := scalar(expression, map[string]*ast.StructType{}, map[string]bool{})
			if got["type"] != "object" || !reflect.DeepEqual(got["additionalProperties"], tt.want) {
				t.Fatalf("metadata schema = %#v; expected object values %#v", got, tt.want)
			}
		})
	}
}
