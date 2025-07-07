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

	"entgo.io/ent/entc/gen"
	"github.com/bufbuild/protocompile/protoutil"
	"github.com/jhump/protoreflect/v2/protobuilder"
	"github.com/jhump/protoreflect/v2/sourceinfo"
	"github.com/yoshino-s/entproto/annotations"
	"github.com/yoshino-s/entproto/convert"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	_ "google.golang.org/protobuf/types/known/emptypb"
	_ "google.golang.org/protobuf/types/known/structpb"
	_ "google.golang.org/protobuf/types/known/timestamppb"
	_ "google.golang.org/protobuf/types/known/wrapperspb" // needed to load wkt to global proto registry
)

const (
	DefaultProtoPackageName = "entpb"
)

var (
	repeatedFieldLabel = descriptorpb.FieldDescriptorProto_LABEL_REPEATED
	wktsPaths          = map[string]string{
		// TODO: handle more Well-Known proto types
		"google.protobuf.Timestamp":   "google/protobuf/timestamp.proto",
		"google.protobuf.Empty":       "google/protobuf/empty.proto",
		"google.protobuf.Int32Value":  "google/protobuf/wrappers.proto",
		"google.protobuf.Int64Value":  "google/protobuf/wrappers.proto",
		"google.protobuf.UInt32Value": "google/protobuf/wrappers.proto",
		"google.protobuf.UInt64Value": "google/protobuf/wrappers.proto",
		"google.protobuf.FloatValue":  "google/protobuf/wrappers.proto",
		"google.protobuf.DoubleValue": "google/protobuf/wrappers.proto",
		"google.protobuf.StringValue": "google/protobuf/wrappers.proto",
		"google.protobuf.BoolValue":   "google/protobuf/wrappers.proto",
		"google.protobuf.BytesValue":  "google/protobuf/wrappers.proto",
		"google.protobuf.Struct":      "google/protobuf/struct.proto",
		"google.protobuf.ListValue":   "google/protobuf/struct.proto",
		"google.protobuf.Value":       "google/protobuf/struct.proto",
	}
	wktsPathsList = make([]string, 0, len(wktsPaths))
)

func init() {
	for _, v := range wktsPaths {
		wktsPathsList = append(wktsPathsList, v)
	}
}

// LoadAdapter takes a *gen.Graph and parses it into protobuf file descriptors
func LoadAdapter(graph *gen.Graph) (*Adapter, error) {
	a := &Adapter{
		graph:            graph,
		descriptors:      make(map[string]protoreflect.FileDescriptor),
		schemaProtoFiles: make(map[string]string),
		errors:           make(map[string]error),

		converters: make(map[*gen.Type]*convert.Converter),
	}
	if err := a.parse(); err != nil {
		return nil, err
	}
	return a, nil
}

// Adapter facilitates the transformation of ent gen.Type to desc.FileDescriptors
type Adapter struct {
	graph            *gen.Graph
	descriptors      map[string]protoreflect.FileDescriptor
	schemaProtoFiles map[string]string
	errors           map[string]error

	converters map[*gen.Type]*convert.Converter
}

// AllFileDescriptors returns a file descriptor per proto package for each package that contains
// a successfully parsed ent.Schema
func (a *Adapter) AllFileDescriptors() map[string]protoreflect.FileDescriptor {
	return a.descriptors
}

// GetMessageDescriptor retrieves the protobuf message descriptor for `schemaName`, if an error was returned
// while trying to parse that error they are returned
func (a *Adapter) GetMessageDescriptor(schemaName string) (protoreflect.MessageDescriptor, error) {
	fd, err := a.GetFileDescriptor(schemaName)
	if err != nil {
		return nil, err
	}
	findMessage := fd.Messages().ByName(protoreflect.Name(schemaName))
	if findMessage != nil {
		return findMessage, nil
	}
	return nil, errors.New("entproto: couldnt find message descriptor")
}

// parse transforms the ent gen.Type objects into file descriptors
func (a *Adapter) parse() error {
	var dpbDescriptors []*descriptorpb.FileDescriptorProto

	protoPackages := make(map[string]*descriptorpb.FileDescriptorProto)

	for _, genType := range a.graph.Nodes {
		protoPkg, err := protoPackageName(genType)
		if err != nil {
			a.errors[genType.Name] = err
			fmt.Fprintln(os.Stderr, "Skipping schema:", genType.Name, "due to proto package name error:", err)
			continue
		}

		if _, ok := protoPackages[protoPkg]; !ok {
			goPkg := a.goPackageName(protoPkg)
			protoPackages[protoPkg] = &descriptorpb.FileDescriptorProto{
				Name:    relFileName(protoPkg),
				Package: &protoPkg,
				Syntax:  strptr("proto3"),
				Options: &descriptorpb.FileOptions{
					GoPackage: &goPkg,
				},
			}
		}
		fd := protoPackages[protoPkg]
		converter := convert.New(fd)
		a.converters[genType] = converter

		messageDescriptor, err := converter.EntTypeToDescriptorProto(a.graph, genType)

		// store specific message parse failures
		if err != nil {
			a.errors[genType.Name] = err
			fmt.Fprintln(os.Stderr, "Skipping schema:", genType.Name, "due to error:", err)
			continue
		}

		fd.MessageType = append(fd.MessageType, messageDescriptor...)
		a.schemaProtoFiles[genType.Name] = *fd.Name

		depPaths, err := a.extractDepPaths(messageDescriptor[0])
		if err != nil {
			a.errors[genType.Name] = err
			fmt.Fprintln(os.Stderr, "Skipping schema:", genType.Name, "due to dependency extraction error:", err)
			continue
		}
		fd.Dependency = append(fd.Dependency, depPaths...)

		svcAnnotation, err := extractServiceAnnotation(genType)
		if errors.Is(err, errNoServiceDef) {
			continue
		}
		if err != nil {
			return err
		}
		if svcAnnotation.Generate {
			svcResources, err := a.createServiceResources(genType, svcAnnotation.Methods)
			if err != nil {
				return err
			}
			fd.Service = append(fd.Service, svcResources.svc)
			fd.MessageType = append(fd.MessageType, svcResources.svcMessages...)
			fd.Dependency = append(fd.Dependency, "google/protobuf/empty.proto")
			fd.Dependency = append(fd.Dependency, "google/protobuf/wrappers.proto")
			fd.Dependency = append(fd.Dependency, "google/protobuf/struct.proto")
		}
	}

	// Append the well known types to the context.
	for _, wktPath := range wktsPaths {
		typeDesc, err := sourceinfo.Files.FindFileByPath(wktPath)
		if err != nil {
			return err
		}
		dpbDescriptors = append(dpbDescriptors, protoutil.ProtoFromFileDescriptor(typeDesc))
	}

	for _, fd := range protoPackages {
		fd.Dependency = dedupe(fd.Dependency)
		dpbDescriptors = append(dpbDescriptors, fd)
	}

	descriptors := make(map[string]protoreflect.FileDescriptor)
	dpbDescriptors = dedupeFileDescriptors(dpbDescriptors)

	fds, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{
		File: dpbDescriptors,
	})
	if err != nil {
		return fmt.Errorf("entproto: failed creating file descriptors: %w", err)
	}

	var rangeErr error
	fds.RangeFiles(func(fd protoreflect.FileDescriptor) bool {

		fbuild, err := protobuilder.FromFile(fd)
		if err != nil {
			rangeErr = fmt.Errorf("entproto: failed building file descriptor for %q: %w", fd.Path(), err)
			return false
		}
		fbuild.SetSyntaxComments(protobuilder.Comments{
			LeadingComment: " Code generated by entproto. DO NOT EDIT.",
		})
		fd, err = fbuild.Build()
		if err != nil {
			rangeErr = fmt.Errorf("entproto: failed building file descriptor for %q: %w", fd.Path(), err)
			return false
		}
		descriptors[string(fd.Path())] = fd
		return true
	})
	if rangeErr != nil {
		return rangeErr
	}

	for _, wktPath := range wktsPathsList {
		delete(descriptors, wktPath)
	}

	a.descriptors = descriptors

	return nil
}

func (a *Adapter) goPackageName(protoPkgName string) string {
	// TODO(rotemtam): make this configurable from an annotation
	entBase := a.graph.Config.Package
	slashed := strings.ReplaceAll(protoPkgName, ".", "/")
	return path.Join(entBase, "proto", slashed)
}

// GetFileDescriptor returns the proto file descriptor containing the transformed proto message descriptor for
// `schemaName` along with any other messages in the same protobuf package.
func (a *Adapter) GetFileDescriptor(schemaName string) (protoreflect.FileDescriptor, error) {
	if err, ok := a.errors[schemaName]; ok {
		return nil, err
	}
	fn, ok := a.schemaProtoFiles[schemaName]
	if !ok {
		return nil, fmt.Errorf("entproto: could not find schemaProtoFiles for schema %s", schemaName)
	}

	dsc, ok := a.descriptors[fn]
	if !ok {
		return nil, fmt.Errorf("entproto: could not find file descriptor for schema %s", schemaName)
	}

	return dsc, nil
}

func protoPackageName(genType *gen.Type) (string, error) {
	msgAnnot, err := annotations.ExtractMessageAnnotation(genType)
	if err != nil {
		return "", err
	}

	if msgAnnot.Package != "" {
		return msgAnnot.Package, nil
	}
	return DefaultProtoPackageName, nil
}

func relFileName(packageName string) *string {
	parts := strings.Split(packageName, ".")
	fileName := parts[len(parts)-1] + ".proto"
	parts = append(parts, fileName)
	joined := filepath.Join(parts...)
	return &joined
}

func (a *Adapter) extractDepPaths(m *descriptorpb.DescriptorProto) ([]string, error) {
	var out []string
	for _, fld := range m.Field {
		if *fld.Type == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
			fieldTypeName := *fld.TypeName
			if wp, ok := wktsPaths[fieldTypeName]; ok {
				out = append(out, wp)
			} else if graphContainsDependency(a.graph, fieldTypeName) {
				fieldTypeName = extractLastFqnPart(fieldTypeName)
				depType, err := extractGenTypeByName(a.graph, fieldTypeName)
				if err != nil {
					return nil, err
				}
				depPackageName, err := protoPackageName(depType)
				if err != nil {
					return nil, err
				}
				selfType, err := extractGenTypeByName(a.graph, *m.Name)
				if err != nil {
					return nil, err
				}
				selfPackageName, _ := protoPackageName(selfType)
				if depPackageName != selfPackageName {
					importPath := relFileName(depPackageName)
					out = append(out, *importPath)
				}
			} else {
				return nil, fmt.Errorf("entproto: failed extracting deps, unknown path for %s", fieldTypeName)
			}
		}
	}
	return out, nil
}

func graphContainsDependency(graph *gen.Graph, fieldTypeName string) bool {
	gt, err := extractGenTypeByName(graph, extractLastFqnPart(fieldTypeName))
	if err != nil {
		return false
	}
	return gt != nil
}

func extractLastFqnPart(fqn string) string {
	parts := strings.Split(fqn, ".")
	return parts[len(parts)-1]
}

func strptr(s string) *string {
	return &s
}

func int32ptr(i int32) *int32 {
	return &i
}

func extractGenTypeByName(graph *gen.Graph, name string) (*gen.Type, error) {
	for _, sch := range graph.Nodes {
		if sch.Name == name {
			return sch, nil
		}
	}
	return nil, fmt.Errorf("entproto: could not find schema %q in graph", name)
}

func dedupe(s []string) []string {
	out := make([]string, 0, len(s))
	seen := make(map[string]struct{})
	for _, item := range s {
		if _, skip := seen[item]; skip {
			continue
		}
		out = append(out, item)
		seen[item] = struct{}{}
	}
	return out
}

func dedupeFileDescriptors(fds []*descriptorpb.FileDescriptorProto) []*descriptorpb.FileDescriptorProto {
	seen := make(map[string]struct{})
	var out []*descriptorpb.FileDescriptorProto
	for _, fd := range fds {
		if _, ok := seen[fd.GetName()]; !ok {
			out = append(out, fd)
			seen[fd.GetName()] = struct{}{}
		}
	}
	return out
}
