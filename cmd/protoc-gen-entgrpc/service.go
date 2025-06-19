package main

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"text/template"

	"entgo.io/ent/entc/gen"
	entFieldPkg "entgo.io/ent/schema/field"
	"github.com/yoshino-s/entproto"
	"google.golang.org/protobuf/compiler/protogen"
)

func newServiceGenerator(plugin *protogen.Plugin, file *protogen.File, graph *gen.Graph, adapter *entproto.Adapter, service *protogen.Service, goImportPath protogen.GoImportPath) (*serviceGenerator, error) {
	typ, err := extractEntTypeName(service, graph)
	if err != nil {
		return nil, err
	}

	filename := file.GeneratedFilenamePrefix + "." + snake(service.GoName) + ".go"
	g := plugin.NewGeneratedFile(filename, goImportPath)
	g.Import(file.GoImportPath)

	fieldMap, err := adapter.FieldMap(typ.Name)
	if err != nil {
		return nil, err
	}
	service.GoName += "Handler"

	return &serviceGenerator{
		GoImportPath:   goImportPath,
		GeneratedFile:  g,
		RuntimePackage: runtimePackage,
		EntPackage:     protogen.GoImportPath(graph.Config.Package),
		ConnectPackage: connectPackage,
		File:           file,
		Service:        service,
		EntType:        typ,
		FieldMap:       fieldMap,
	}, nil
}

func (g *serviceGenerator) generate() error {
	tmpl, err := gen.NewTemplate("service").
		Funcs(template.FuncMap{
			"debug": func(v any) string {
				fmt.Fprintf(os.Stderr, "// DEBUG: %v\n", v)
				return ""
			},
			"ident":    g.QualifiedGoIdent,
			"entIdent": g.entIdent,
			"unquote":  func(v any) (string, error) { return strconv.Unquote(fmt.Sprint(v)) },
			"qualify": func(pkg, ident string) string {
				return g.QualifiedGoIdent(protogen.GoImportPath(pkg).Ident(ident))
			},
			"removeSuffix": func(s, suffix string) string {
				if strings.HasSuffix(s, suffix) {
					return s[:len(s)-len(suffix)]
				}
				return s
			},
			"newConverter":        g.newConverter,
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
			"method": func(m *protogen.Method) *methodInput {
				return &methodInput{
					G:      g,
					Method: m,
				}
			},
			"callHook": func(action string, query string) string {
				return fmt.Sprintf(
					"if err := svc.RunHooks(ctx, %s, req, %s); err != nil { return nil, err }",
					g.QualifiedGoIdent(runtimePackage.Ident("Action"+action)),
					query,
				)
			},
			"callHook3": func(action string, query string) string {
				return fmt.Sprintf(
					"if err := svc.RunHooks(ctx, %s, req, %s); err != nil { return nil, nil, err }",
					g.QualifiedGoIdent(runtimePackage.Ident("Action"+action)),
					query,
				)
			},
			"hasSuffix": func(s, suffix string) bool {
				return strings.HasSuffix(s, suffix)
			},
			"getFilters": func(m *methodInput) []*filterField {
				for _, field := range m.Method.Input.Fields {
					if field.Desc.Name() == "filter" {
						fields := []*filterField{}

						mm := map[string]*gen.Field{}
						for _, f := range m.G.EntType.Fields {
							mm[f.Name] = f
						}

						for _, f := range field.Message.Fields {
							if strings.HasSuffix(string(f.Desc.Name()), "_in") {
								name := strings.TrimSuffix(string(f.Desc.Name()), "_in")
								if entField, ok := mm[name]; ok {
									typ := entField.Type.Type.String()
									if entField.Type.Type == entFieldPkg.TypeTime {
										typ = g.QualifiedGoIdent(protogen.GoImportPath("time").Ident("Time"))
									} else if entField.Type.Type == entFieldPkg.TypeJSON {
										typ = g.QualifiedGoIdent(protogen.GoImportPath("encoding/json").Ident("RawMessage"))
									}
									fields = append(fields, &filterField{
										Field: &entproto.FieldMappingDescriptor{
											EntField:          entField,
											PbFieldDescriptor: f.Desc,
										},
										Operation: fmt.Sprintf("%sIn", entField.StructField()),
										Optional:  entField.Type.Type != entFieldPkg.TypeEnum,
										Type:      typ,
									})
								}
							} else if strings.HasSuffix(string(f.Desc.Name()), "_contains") {
								name := strings.TrimSuffix(string(f.Desc.Name()), "_contains")
								if entField, ok := mm[name]; ok {
									fields = append(fields, &filterField{
										Field: &entproto.FieldMappingDescriptor{
											EntField:          entField,
											PbFieldDescriptor: f.Desc,
										},
										Operation: fmt.Sprintf("%sContains", entField.StructField()),
										Optional:  entField.Type.Type != entFieldPkg.TypeEnum,
									})
								}
							} else {
								name := string(f.Desc.Name())
								if entField, ok := mm[name]; ok {
									fields = append(fields, &filterField{
										Field: &entproto.FieldMappingDescriptor{
											EntField:          entField,
											PbFieldDescriptor: f.Desc,
										},
										Operation: fmt.Sprintf("%sEQ", entField.StructField()),
										Optional:  entField.Type.Type != entFieldPkg.TypeEnum,
									})
								}
							}
						}

						return fields
					}
				}
				return nil
			},
		}).
		ParseFS(templates, "template/service/*.tmpl")
	if err != nil {
		return err
	}
	if err := tmpl.ExecuteTemplate(g, "service", g); err != nil {
		return fmt.Errorf("template execution failed: %w", err)
	}
	return nil
}

type (
	serviceGenerator struct {
		*protogen.GeneratedFile
		RuntimePackage protogen.GoImportPath
		GoImportPath   protogen.GoImportPath
		EntPackage     protogen.GoImportPath
		ConnectPackage protogen.GoImportPath
		File           *protogen.File
		Service        *protogen.Service
		EntType        *gen.Type
		FieldMap       entproto.FieldMap
	}
	methodInput struct {
		G      *serviceGenerator
		Method *protogen.Method
	}
	filterField struct {
		Field     *entproto.FieldMappingDescriptor
		Operation string
		Optional  bool
		Type      string
	}
)

func extractEntTypeName(s *protogen.Service, g *gen.Graph) (*gen.Type, error) {
	typeName := strings.TrimSuffix(s.GoName, "Service")
	for _, gt := range g.Nodes {
		if gt.Name == typeName {
			return gt, nil
		}
	}
	return nil, fmt.Errorf("entproto: type %q of service %q not found in graph", typeName, s.GoName)
}

func (g *serviceGenerator) entIdent(subpath string, ident string) protogen.GoIdent {
	ip := path.Join(string(g.EntPackage), subpath)
	return protogen.GoImportPath(ip).Ident(ident)
}
