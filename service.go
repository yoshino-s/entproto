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

	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"github.com/go-viper/mapstructure/v2"
	"google.golang.org/protobuf/types/descriptorpb"
	_ "google.golang.org/protobuf/types/known/emptypb"
)

const (
	ServiceAnnotation = "ProtoService"
	// MaxPageSize is the maximum page size that can be returned by a List call. Requesting page sizes larger than
	// this value will return, at most, MaxPageSize entries.
	MaxPageSize = 1000
	// MaxBatchCreateSize is the maximum number of entries that can be created by a single BatchCreate call. Requests
	// exceeding this batch size will return an error.
	MaxBatchCreateSize = 1000
	// MethodCreate generates a Create gRPC service method for the entproto.Service.
	MethodCreate Method = 1 << iota
	// MethodGet generates a Get gRPC service method for the entproto.Service.
	MethodGet
	// MethodUpdate generates an Update gRPC service method for the entproto.Service.
	MethodUpdate
	// MethodDelete generates a Delete gRPC service method for the entproto.Service.
	MethodDelete
	// MethodList generates a List gRPC service method for the entproto.Service.
	MethodList
	// MethodAll generates all service methods for the entproto.Service. This is the same behavior as not including entproto.Methods.
	MethodAll = MethodCreate | MethodGet | MethodUpdate | MethodDelete | MethodList
)

var (
	errNoServiceDef = errors.New("entproto: annotation entproto.Service missing")
)

type Method uint

// Is reports whether method m matches given method n.
func (m Method) Is(n Method) bool { return m&n != 0 }

// Methods specifies the gRPC service methods to generate for the entproto.Service.
func Methods(methods Method) ServiceOption {
	return func(s *service) {
		s.Methods = methods
	}
}

type service struct {
	Generate bool
	Methods  Method
}

func (service) Name() string {
	return ServiceAnnotation
}

// ServiceOption configures the entproto.Service annotation.
type ServiceOption func(svc *service)

// Service annotates an ent.Schema to specify that protobuf service generation is required for it.
func Service(opts ...ServiceOption) schema.Annotation {
	s := service{
		Generate: true,
	}
	for _, apply := range opts {
		apply(&s)
	}
	// Default to generating all methods.
	if s.Methods == 0 {
		s.Methods = MethodAll
	}
	return s
}

func (a *Adapter) createServiceResources(genType *gen.Type, methods Method) (serviceResources, error) {
	name := genType.Name
	serviceFqn := fmt.Sprintf("%sService", name)

	out := serviceResources{
		svc: &descriptorpb.ServiceDescriptorProto{
			Name: &serviceFqn,
		},
	}

	for _, m := range []Method{MethodCreate, MethodGet, MethodUpdate, MethodDelete, MethodList} {
		if !methods.Is(m) {
			continue
		}

		resources, err := a.genMethodProtos(genType, m)
		if err != nil {
			return serviceResources{}, err
		}
		out.svc.Method = append(out.svc.Method, resources.methodDescriptor)
		out.svcMessages = append(out.svcMessages, resources.messages...)
	}
	out.svcMessages = dedupeServiceMessages(out.svcMessages)

	return out, nil
}

func (a *Adapter) genMethodProtos(genType *gen.Type, m Method) (methodResources, error) {
	input := &descriptorpb.DescriptorProto{}
	protoMessageFieldType := descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
	repeatedFieldLabel := descriptorpb.FieldDescriptorProto_LABEL_REPEATED
	noSideEffectIdempotencyLevel := descriptorpb.MethodOptions_NO_SIDE_EFFECTS
	messages := []*descriptorpb.DescriptorProto{}
	method := &descriptorpb.MethodDescriptorProto{}

	switch m {
	case MethodGet:
		method.Name = strptr("Get")
		method.InputType = strptr("google.protobuf.Int32Value")
		method.OutputType = strptr(genType.Name)
		method.Options = &descriptorpb.MethodOptions{
			IdempotencyLevel: &noSideEffectIdempotencyLevel,
		}
	case MethodCreate:
		method.Name = strptr("Create")
		method.InputType = strptr(genType.Name)
		method.OutputType = strptr(genType.Name)
	case MethodUpdate:
		method.Name = strptr("Update")
		method.InputType = strptr(genType.Name)
		method.OutputType = strptr(genType.Name)
	case MethodDelete:
		method.Name = strptr("Delete")
		method.InputType = strptr("google.protobuf.Int32Value")
		method.OutputType = strptr("google.protobuf.Empty")
	case MethodList:
		if !(genType.ID.Type.Type.Integer() || genType.ID.IsUUID() || genType.ID.IsString()) {
			return methodResources{}, fmt.Errorf("entproto: list method does not support schema %q id type %q",
				genType.Name, genType.ID.Type.String())
		}
		int32FieldType := descriptorpb.FieldDescriptorProto_TYPE_INT32

		method.Name = strptr("List")
		method.InputType = strptr(fmt.Sprintf("List%sRequest", genType.Name))
		method.Options = &descriptorpb.MethodOptions{
			IdempotencyLevel: &noSideEffectIdempotencyLevel,
		}

		boolFieldType := descriptorpb.FieldDescriptorProto_TYPE_BOOL
		filterMessage := &descriptorpb.DescriptorProto{
			Name:  strptr(fmt.Sprintf("List%sFilter", genType.Name)),
			Field: []*descriptorpb.FieldDescriptorProto{},
		}

		input.Name = method.InputType
		input.Field = []*descriptorpb.FieldDescriptorProto{
			{
				Name:     strptr("offset"),
				Number:   int32ptr(1),
				Type:     &protoMessageFieldType,
				TypeName: strptr("google.protobuf.Int32Value"),
			},
			{
				Name:     strptr("limit"),
				Number:   int32ptr(2),
				Type:     &protoMessageFieldType,
				TypeName: strptr("google.protobuf.Int32Value"),
			},
			{
				Name:     strptr("order"),
				Number:   int32ptr(3),
				Type:     &protoMessageFieldType,
				TypeName: strptr("google.protobuf.StringValue"),
			},
			{
				Name:   strptr("descending"),
				Number: int32ptr(4),
				Type:   &boolFieldType,
			},
			{
				Name:     strptr("filter"),
				Number:   int32ptr(5),
				Type:     &protoMessageFieldType,
				TypeName: strptr(filterMessage.GetName()),
			},
			{
				Name:   strptr("no_limit"),
				Number: int32ptr(6),
				Type:   &boolFieldType,
			},
		}

		for _, genField := range genType.Fields {
			filterAnnotation, err := extractFilterAnnotation(genField)
			if err != nil {
				return methodResources{}, fmt.Errorf("entproto: unable to decode entproto.Filter annotation for schema %q field %q: %w",
					genType.Name, genField.Name, err)
			}

			if filterAnnotation != nil {
				oldOptional := genField.Optional
				if genField.Type.Type != field.TypeEnum {
					genField.Optional = true
				}
				optionalFieldType, err := extractProtoTypeDetails(genField)
				genField.Optional = oldOptional
				if err != nil {
					return methodResources{}, fmt.Errorf("entproto: unable to extract proto type details for schema %q field %q: %w",
						genType.Name, genField.Name, err)
				}

				originalFieldType, err := extractProtoTypeDetails(genField)
				if err != nil {
					return methodResources{}, fmt.Errorf("entproto: unable to extract proto type details for schema %q field %q: %w",
						genType.Name, genField.Name, err)
				}

				if genField.Type.Type == field.TypeEnum {
					optionalFieldType.messageName = fmt.Sprintf("%s.%s", genType.Name, optionalFieldType.messageName)
					originalFieldType.messageName = fmt.Sprintf("%s.%s", genType.Name, originalFieldType.messageName)
				}
				if filterAnnotation.Mode&FilterModeEQ != 0 {
					filterMessage.Field = append(filterMessage.Field, &descriptorpb.FieldDescriptorProto{
						Name:     strptr(snake(genField.Name)),
						Number:   int32ptr(int32(len(filterMessage.Field) + 1)),
						Type:     &optionalFieldType.protoType,
						TypeName: strptr(optionalFieldType.messageName),
					})
				}
				if filterAnnotation.Mode&FilterModeContains != 0 {
					if genField.Type.Type != field.TypeString {
						return methodResources{}, fmt.Errorf("entproto: contains filter mode is only supported for string fields, schema %q field %q has type %q",
							genType.Name, genField.Name, genField.Type.Type)
					}
					filterMessage.Field = append(filterMessage.Field, &descriptorpb.FieldDescriptorProto{
						Name:     strptr(fmt.Sprintf("%s_contains", snake(genField.Name))),
						Number:   int32ptr(int32(len(filterMessage.Field) + 1)),
						Type:     &optionalFieldType.protoType,
						TypeName: strptr(optionalFieldType.messageName),
					})
				}
				if filterAnnotation.Mode&FilterModeIn != 0 {
					filterMessage.Field = append(filterMessage.Field, &descriptorpb.FieldDescriptorProto{
						Name:     strptr(fmt.Sprintf("%s_in", snake(genField.Name))),
						Number:   int32ptr(int32(len(filterMessage.Field) + 1)),
						Type:     &originalFieldType.protoType,
						TypeName: strptr(originalFieldType.messageName),
						Label:    &repeatedFieldLabel,
					})
				}
			}
		}

		extraFilterAnnotation, err := extractExtraFilterAnnotation(genType)
		if err != nil {
			return methodResources{}, fmt.Errorf("entproto: unable to decode entproto.ExtraFilter annotation for schema %q: %w",
				genType.Name, err)
		}
		if extraFilterAnnotation != nil {
			for name, fieldType := range extraFilterAnnotation.ExtraFields {
				filterMessage.Field = append(filterMessage.Field, &descriptorpb.FieldDescriptorProto{
					Name:     strptr(snake(name)),
					Number:   int32ptr(int32(len(filterMessage.Field) + 1)),
					Type:     &protoMessageFieldType,
					TypeName: strptr(typeMap[fieldType].optionalType),
				})
			}
		}

		method.OutputType = strptr(fmt.Sprintf("List%sResponse", genType.Name))
		output := &descriptorpb.DescriptorProto{
			Name: method.OutputType,
			Field: []*descriptorpb.FieldDescriptorProto{
				{
					Name:     strptr("items"),
					Number:   int32ptr(1),
					Label:    &repeatedFieldLabel,
					Type:     &protoMessageFieldType,
					TypeName: strptr(genType.Name),
				},
				{
					Name:   strptr("total"),
					Number: int32ptr(2),
					Type:   &int32FieldType,
				},
			},
		}
		messages = append(messages, filterMessage, input, output)
	default:
		return methodResources{}, fmt.Errorf("unknown method %q", m)
	}
	return methodResources{
		methodDescriptor: method,
		messages:         messages,
	}, nil
}

type methodResources struct {
	methodDescriptor *descriptorpb.MethodDescriptorProto
	messages         []*descriptorpb.DescriptorProto
}

type serviceResources struct {
	svc         *descriptorpb.ServiceDescriptorProto
	svcMessages []*descriptorpb.DescriptorProto
}

func extractServiceAnnotation(sch *gen.Type) (*service, error) {
	annot, ok := sch.Annotations[ServiceAnnotation]
	if !ok {
		return nil, fmt.Errorf("%w: entproto: schema %q does not have an entproto.Service annotation",
			errNoServiceDef, sch.Name)
	}

	var out service
	err := mapstructure.Decode(annot, &out)
	if err != nil {
		return nil, fmt.Errorf("entproto: unable to decode entproto.Service annotation for schema %q: %w",
			sch.Name, err)
	}

	return &out, nil
}

func dedupeServiceMessages(msgs []*descriptorpb.DescriptorProto) []*descriptorpb.DescriptorProto {
	out := make([]*descriptorpb.DescriptorProto, 0, len(msgs))
	seen := make(map[string]struct{})
	for _, msg := range msgs {
		if _, skip := seen[msg.GetName()]; skip {
			continue
		}
		out = append(out, msg)
		seen[msg.GetName()] = struct{}{}
	}
	return out
}
