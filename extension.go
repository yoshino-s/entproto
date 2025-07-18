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

package entproto

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/jhump/protoreflect/v2/protoprint"
	"github.com/yoshino-s/entproto/convert"
	"go.uber.org/multierr"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ExtensionOption is an option for the entproto extension.
type ExtensionOption func(*Extension)

// NewExtension returns a new Extension configured by opts.
func NewExtension(opts ...ExtensionOption) (*Extension, error) {
	e := &Extension{}
	for _, opt := range opts {
		opt(e)
	}
	return e, nil
}

// Extension is an entc.Extension that generates .proto files from an ent schema.
// To use within an entc.go file:
//
//	func main() {
//		if err := entc.Generate("./schema",
//			&gen.Config{},
//			entc.Extensions(
//				entproto.NewExtension(),
//			),
//		); err != nil {
//			log.Fatal("running ent codegen:", err)
//		}
//	}
type Extension struct {
	entc.DefaultExtension
	protoDir string
}

// WithProtoDir sets the directory where the generated .proto files will be written.
func WithProtoDir(dir string) ExtensionOption {
	return func(e *Extension) {
		e.protoDir = dir
	}
}

// Hooks implements entc.Extension.
func (e *Extension) Hooks() []gen.Hook {
	return []gen.Hook{e.hook()}
}

func (e *Extension) hook() gen.Hook {
	return func(next gen.Generator) gen.Generator {
		return gen.GenerateFunc(func(g *gen.Graph) error {
			// Because Generate has side effects (it is writing to the filesystem under gen.Config.Target),
			// we first run all generators, and only then invoke our code. This isn't great, and there's an
			// [open issue](https://github.com/ent/ent/issues/1311) to support this use-case better.
			err := next.Generate(g)
			if err != nil {
				return err
			}
			return e.generate(g)
		})
	}
}

// Hook returns a gen.Hook that invokes Generate.
// To use it pragmatically:
//
//	entc.Generate("./ent/schema", &gen.Config{
//	  Hooks: []gen.Hook{
//	    entproto.Hook(),
//	  },
//	})
//
// Deprecated: use Extension instead.
func Hook() gen.Hook {
	x := &Extension{}
	return x.hook()
}

// Generate takes a *gen.Graph and creates .proto files.
// Next to each .proto file, Generate creates a generate.go
// file containing a //go:generate directive to invoke protoc and compile Go code from the protobuf definitions.
// If generate.go already exists next to the .proto file, this step is skipped.
func Generate(g *gen.Graph) error {
	x := &Extension{}
	return x.generate(g)
}

func (e *Extension) generate(g *gen.Graph) error {
	entProtoDir := path.Join(g.Config.Target, "proto")
	if e.protoDir != "" {
		entProtoDir = e.protoDir
	}
	adapter, err := LoadAdapter(g)
	if err != nil {
		return fmt.Errorf("entproto: failed parsing ent graph: %w", err)
	}
	var errs error
	for _, schema := range g.Schemas {
		name := schema.Name
		_, err := adapter.GetFileDescriptor(name)
		if err != nil && !errors.Is(err, convert.ErrSchemaSkipped) {
			errs = multierr.Append(errs, err)
		}
	}
	if errs != nil {
		return fmt.Errorf("entproto: failed parsing some schemas: %w", errs)
	}
	allDescriptors := make([]protoreflect.FileDescriptor, 0, len(adapter.AllFileDescriptors()))
	for _, filedesc := range adapter.AllFileDescriptors() {
		allDescriptors = append(allDescriptors, filedesc)
	}
	// Print the .proto files.
	printer := &protoprint.Printer{}
	if err = printer.PrintProtosToFileSystem(allDescriptors, entProtoDir); err != nil {
		return fmt.Errorf("entproto: failed writing .proto files: %w", err)
	}
	return nil
}

func fileExists(fpath string) bool {
	if _, err := os.Stat(fpath); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func protocGenerateGo(fd protoreflect.FileDescriptor, toSchemaDir string) string {
	levelsUp := len(strings.Split(string(fd.Package()), "."))
	toProtoBase := ""
	for i := 0; i < levelsUp; i++ {
		toProtoBase = filepath.Join("..", toProtoBase)
	}
	protocCmd := []string{
		"protoc",
		"-I=" + toProtoBase,
		"--go_out=" + toProtoBase,
		"--go-grpc_out=" + toProtoBase,
		"--go_opt=paths=source_relative",
		"--go-grpc_opt=paths=source_relative",
		"--entgrpc_out=" + toProtoBase,
		"--entgrpc_opt=paths=source_relative,schema_path=" + toSchemaDir,
		string(fd.Name()),
	}
	goGen := fmt.Sprintf("//go:generate %s", strings.Join(protocCmd, " "))
	goPkgName := extractLastFqnPart(string(fd.Package()))
	return fmt.Sprintf("package %s\n%s\n", goPkgName, goGen)
}
