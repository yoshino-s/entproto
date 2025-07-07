// Copyright 2019-present Facebook
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"embed"
	"errors"
	"flag"
	"path"
	"path/filepath"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/yoshino-s/entproto"
	"github.com/yoshino-s/entproto/convert"
	"google.golang.org/protobuf/compiler/protogen"
)

var (
	entSchemaPath  *string
	snake          = gen.Funcs["snake"].(func(string) string)
	connectPackage = protogen.GoImportPath("connectrpc.com/connect")
	runtimePackage = protogen.GoImportPath("github.com/yoshino-s/entproto/runtime")
)

func main() {
	var flags flag.FlagSet
	entSchemaPath = flags.String("schema_path", "", "ent schema path")
	protogen.Options{
		ParamFunc: flags.Set,
	}.Run(func(plg *protogen.Plugin) error {
		g, err := entc.LoadGraph(*entSchemaPath, &gen.Config{})
		if err != nil {
			return err
		}
		for _, f := range plg.Files {
			if !f.Generate {
				continue
			}
			if err := processFile(plg, f, g); err != nil {
				return err
			}
		}
		return nil
	})
}

// processFile generates service implementations from all services defined in the file.
func processFile(gen *protogen.Plugin, file *protogen.File, graph *gen.Graph) error {
	adapter, err := entproto.LoadAdapter(graph)
	if err != nil {
		return err
	}
	file.GoPackageName += protogen.GoPackageName("service")
	generatedFilenamePrefixToSlash := filepath.ToSlash(file.GeneratedFilenamePrefix)
	file.GeneratedFilenamePrefix = path.Join(
		path.Dir(generatedFilenamePrefixToSlash),
		string(file.GoPackageName),
		path.Base(generatedFilenamePrefixToSlash),
	)
	goImportPath := protogen.GoImportPath(path.Join(
		string(file.GoImportPath),
		string(file.GoPackageName),
	))

	for _, m := range file.Messages {
		if _, err := extractEntTypeNameFromMessage(m, graph); err != nil {
			continue
		}
		sg, err := newMessageGenerator(gen, file, graph, adapter, m, goImportPath)
		if err != nil {
			if errors.Is(err, convert.ErrSchemaSkipped) {
				continue
			}
			return err
		}
		if err := sg.generate(); err != nil {
			return err
		}
	}

	svcGenerated := false
	for _, s := range file.Services {
		if name := string(s.Desc.Name()); !containsSvc(adapter, name) {
			continue
		}
		sg, err := newServiceGenerator(gen, file, graph, adapter, s, goImportPath)
		if err != nil {
			return err
		}
		if err := sg.generate(); err != nil {
			return err
		}
		svcGenerated = true
	}

	if svcGenerated {
		hg, err := newHelperGenerator(gen, file, graph, adapter, goImportPath)
		if err != nil {
			return err
		}
		if err := hg.generate(); err != nil {
			return err
		}
	}

	return nil
}

// containsSvc reports if the service definition for svc is created by the adapter.
func containsSvc(adapter *entproto.Adapter, svc string) bool {
	for _, d := range adapter.AllFileDescriptors() {
		//for _, s := range d.Services() {
		for i := 0; i < d.Services().Len(); i++ {
			s := d.Services().Get(i)
			if string(s.Name()) == svc {
				return true
			}
		}
	}
	return false
}

//go:embed template/*
var templates embed.FS
