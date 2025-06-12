package main

import (
	"fmt"
	"strings"
	"text/template"

	"entgo.io/ent/entc/gen"
	"github.com/yoshino-s/entproto"
	"google.golang.org/protobuf/compiler/protogen"
)

func newHelperGenerator(plugin *protogen.Plugin, file *protogen.File, graph *gen.Graph, adapter *entproto.Adapter, goImportPath protogen.GoImportPath) (*helperGenerator, error) {
	filename := file.GeneratedFilenamePrefix + ".service_helper.go"
	g := plugin.NewGeneratedFile(filename, goImportPath)
	g.Import(file.GoImportPath)

	return &helperGenerator{
		GoImportPath:   goImportPath,
		GeneratedFile:  g,
		RuntimePackage: runtimePackage,
		EntPackage:     protogen.GoImportPath(graph.Config.Package),
		ConnectPackage: connectPackage,
		File:           file,
	}, nil
}

func (g *helperGenerator) generate() error {
	tmpl, err := gen.NewTemplate("helper").
		Funcs(template.FuncMap{
			"ident": g.QualifiedGoIdent,
			"qualify": func(pkg, ident string) string {
				return g.QualifiedGoIdent(protogen.GoImportPath(pkg).Ident(ident))
			},
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
			"callHook": func(action string, query string) string {
				return fmt.Sprintf(
					"if err := svc.RunHooks(ctx, %s, req, %s); err != nil { return nil, err }",
					g.QualifiedGoIdent(runtimePackage.Ident("Action"+action)),
					query,
				)
			},
		}).
		ParseFS(templates, "template/helper/*.tmpl")
	if err != nil {
		return err
	}
	if err := tmpl.ExecuteTemplate(g, "helper", g); err != nil {
		return fmt.Errorf("template execution failed: %w", err)
	}
	return nil
}

type (
	helperGenerator struct {
		*protogen.GeneratedFile
		RuntimePackage protogen.GoImportPath
		GoImportPath   protogen.GoImportPath
		EntPackage     protogen.GoImportPath
		ConnectPackage protogen.GoImportPath
		File           *protogen.File
	}
)
