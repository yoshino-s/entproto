package main

import (
	"entgo.io/ent/entc/gen"
	"github.com/yoshino-s/entproto"
	"google.golang.org/protobuf/compiler/protogen"
)

type generator struct {
	*protogen.GeneratedFile
	EntType  *gen.Type
	FieldMap entproto.FieldMap

	RuntimePackage protogen.GoImportPath
	GoImportPath   protogen.GoImportPath
	EntPackage     protogen.GoImportPath
	ConnectPackage protogen.GoImportPath

	File *protogen.File
}
