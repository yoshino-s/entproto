package main

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/yoshino-s/entproto/convert/struct_converter"
)

func TestTypeToName(t *testing.T) {
	typ, _ := struct_converter.MarshalGoType(reflect.TypeFor[map[string]any]())
	name := typeToName(typ)

	fmt.Println(name)
}
