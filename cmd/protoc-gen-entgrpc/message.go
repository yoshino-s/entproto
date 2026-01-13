package main

import (
	"fmt"
	"path"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"entgo.io/ent/entc/gen"
	"github.com/yoshino-s/entproto"
	"github.com/yoshino-s/entproto/annotations"
	"google.golang.org/protobuf/compiler/protogen"
)

func newMessageGenerator(plugin *protogen.Plugin, file *protogen.File, graph *gen.Graph, adapter *entproto.Adapter, message *protogen.Message, goImportPath protogen.GoImportPath) (*messageGenerator, error) {
	typ, _ := extractEntTypeNameFromMessage(message, graph)

	filename := file.GeneratedFilenamePrefix + "." + snake(typ.Name) + ".go"
	g := plugin.NewGeneratedFile(filename, goImportPath)
	g.Import(file.GoImportPath)

	fieldMap, err := adapter.FieldMap(typ.Name)
	if err != nil {
		return nil, err
	}

	return &messageGenerator{
		generator: &generator{
			GeneratedFile: g,
			EntType:       typ,

			GoImportPath:   goImportPath,
			RuntimePackage: runtimePackage,
			EntPackage:     protogen.GoImportPath(graph.Config.Package),
			ConnectPackage: connectPackage,
			File:           file,
			FieldMap:       fieldMap,
		},
		Message: message,
	}, nil
}

func (g *messageGenerator) generate() error {
	tmpl, err := gen.NewTemplate("method").
		Funcs(template.FuncMap{
			"ident":        g.QualifiedGoIdent,
			"entIdent":     g.entIdent,
			"newConverter": g.newConverter,
			"unquote":      func(v any) (string, error) { return strconv.Unquote(fmt.Sprint(v)) },
			"qualify": func(pkg, ident string) string {
				return g.QualifiedGoIdent(protogen.GoImportPath(pkg).Ident(ident))
			},
			"protoIdentNormalize": entproto.NormalizeEnumIdentifier,
			"statusErr": func(code, msg string) string {
				return fmt.Sprintf("%s(%s, %s(%q))",
					g.QualifiedGoIdent(connectPackage.Ident("NewError")),
					g.QualifiedGoIdent(connectPackage.Ident(code)),
					g.QualifiedGoIdent(protogen.GoImportPath("github.com/go-errors/errors").Ident("New")),
					msg,
				)
			},
			"statusErrf": func(code, format string, args ...string) string {
				return fmt.Sprintf("%s(%s, %s(%q, %s))",
					g.QualifiedGoIdent(connectPackage.Ident("NewError")),
					g.QualifiedGoIdent(connectPackage.Ident(code)),
					g.QualifiedGoIdent(protogen.GoImportPath("github.com/go-errors/errors").Ident("Errorf")),
					format,
					strings.Join(args, ","),
				)
			},
			"refStruct": func(f *gen.Field) string {
				if f.IsJSON() {
					annotation, err := annotations.ExtractFieldAnnotation(f)
					if err != nil || annotation.MarshaledGoType == nil {
						return ""
					}
					if annotation.MarshaledGoType.Kind == reflect.Struct {
						return "&"
					}
				}
				return ""
			},
		}).
		ParseFS(templates, "template/message/*.tmpl")
	if err != nil {
		return err
	}
	if err := tmpl.ExecuteTemplate(g, "message", g); err != nil {
		return fmt.Errorf("template execution failed: %w", err)
	}
	return nil
}

func (g *messageGenerator) t(t string, data map[string]any) string {
	tmpl, err := gen.NewTemplate("method").
		Funcs(template.FuncMap{
			"ident":        g.QualifiedGoIdent,
			"entIdent":     g.entIdent,
			"newConverter": g.newConverter,
			"unquote":      func(v any) (string, error) { return strconv.Unquote(fmt.Sprint(v)) },
			"qualify": func(pkg, ident string) string {
				return g.QualifiedGoIdent(protogen.GoImportPath(pkg).Ident(ident))
			},
			"protoIdentNormalize": entproto.NormalizeEnumIdentifier,
			"statusErr": func(code, msg string) string {
				return fmt.Sprintf("%s(%s, %s(%q))",
					g.QualifiedGoIdent(connectPackage.Ident("NewError")),
					g.QualifiedGoIdent(connectPackage.Ident(code)),
					g.QualifiedGoIdent(protogen.GoImportPath("github.com/go-errors/errors").Ident("New")),
					msg,
				)
			},
			"statusErrf": func(code, format string, args ...string) string {
				return fmt.Sprintf("%s(%s, %s(%q, %s))",
					g.QualifiedGoIdent(connectPackage.Ident("NewError")),
					g.QualifiedGoIdent(connectPackage.Ident(code)),
					g.QualifiedGoIdent(protogen.GoImportPath("github.com/go-errors/errors").Ident("Errorf")),
					format,
					strings.Join(args, ","),
				)
			},
		}).
		Parse(strings.TrimPrefix(t, "\n"))
	if err != nil {
		panic(err)
	}
	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		panic(err)
	}
	return sb.String()
}

type (
	messageGenerator struct {
		*generator
		Message *protogen.Message
	}
)

func (g *messageGenerator) entIdent(subpath string, ident string) protogen.GoIdent {
	ip := path.Join(string(g.EntPackage), subpath)
	return protogen.GoImportPath(ip).Ident(ident)
}

func extractEntTypeNameFromMessage(s *protogen.Message, g *gen.Graph) (*gen.Type, error) {
	typeName := s.GoIdent.GoName
	for _, gt := range g.Nodes {
		if gt.Name == typeName {
			return gt, nil
		}
	}
	return nil, fmt.Errorf("entproto: type %q not found in graph", typeName)
}
